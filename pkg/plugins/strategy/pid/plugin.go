package pid

import (
	"fmt"
	"github.com/Jamesits/nomad-autoscaler-plugins/pkg/utils"
	"maps"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad-autoscaler/plugins/base"
	"github.com/hashicorp/nomad-autoscaler/plugins/strategy"
	"github.com/hashicorp/nomad-autoscaler/sdk"
)

const (
	pluginName = "strategy-pid"

	runConfigKeyTarget                            = "target"
	runConfigKeyKp                                = "proportional_factor"
	runConfigKeyKi                                = "integral_factor"
	runConfigKeyKd                                = "derivative_factor"
	runConfigKeyTimeDividerNanoSec                = "time_divider_ns"
	runConfigKeyActionCountPolynomialCoefficients = "output_coefficients"
	runConfigKeyActionCountQuantification         = "output_quantification"
	runConfigKeyActionCountMax                    = "output_clamp_max"
	runConfigKeyActionCountMin                    = "output_clamp_min"
	runConfigKeyActionCountDeadZone               = "output_dead_zone"
)

var (
	pluginInfo = &base.PluginInfo{
		Name:       pluginName,
		PluginType: sdk.PluginTypeStrategy,
	}

	defaultConfig = map[string]string{
		runConfigKeyTarget:             "0.0",
		runConfigKeyKp:                 "1.0",
		runConfigKeyKi:                 "0.0",
		runConfigKeyKd:                 "0.0",
		runConfigKeyTimeDividerNanoSec: "1000000000",
		runConfigKeyActionCountPolynomialCoefficients: "0.0, 1.0",
		runConfigKeyActionCountQuantification:         "round",
		runConfigKeyActionCountMax:                    "1000.0",
		runConfigKeyActionCountMin:                    "0.0",
		runConfigKeyActionCountDeadZone:               "0",
	}
)

// Test interface compatibility
var _ strategy.Strategy = (*StrategyPlugin)(nil)

type policyState struct {
	// config
	target                      float64
	kp                          float64
	ki                          float64
	kd                          float64
	timeDivider                 time.Duration
	countPolynomialCoefficients []float64
	countQuantification         string
	countMax                    float64
	countMin                    float64
	countDeadZone               int64

	// internal states
	hasPreviousData bool
	previousTime    time.Time
	previousError   float64
	integral        float64
}

type StrategyPlugin struct {
	config map[string]string
	logger hclog.Logger

	states map[string]*policyState
}

func NewPIDPlugin(log hclog.Logger) strategy.Strategy {
	return &StrategyPlugin{
		logger: log,
		states: make(map[string]*policyState),
	}
}

func (s *StrategyPlugin) PluginInfo() (*base.PluginInfo, error) {
	s.logger.Debug("PluginInfo() called")
	return pluginInfo, nil
}

func (s *StrategyPlugin) SetConfig(config map[string]string) error {
	s.logger.Debug("SetConfig() called", "config", config)

	// config override
	s.config = make(map[string]string)
	maps.Copy(s.config, defaultConfig)
	maps.Copy(s.config, config)

	return nil
}

func (s *StrategyPlugin) newPolicy(id string, config map[string]string) (state *policyState, err error) {
	if existingState, ok := s.states[id]; ok {
		return existingState, nil
	}

	// config override
	c := make(map[string]string)
	maps.Copy(c, s.config)
	maps.Copy(c, config)

	s.logger.Debug("creating new policy state", "id", id, "config", c)
	state = &policyState{}

	// parse args
	state.target, err = strconv.ParseFloat(c[runConfigKeyTarget], 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %s: %w", runConfigKeyTarget, err)
	}

	state.kp, err = strconv.ParseFloat(c[runConfigKeyKp], 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %s: %w", runConfigKeyKp, err)
	}

	state.ki, err = strconv.ParseFloat(c[runConfigKeyKi], 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %s: %w", runConfigKeyKi, err)
	}

	state.kd, err = strconv.ParseFloat(c[runConfigKeyKd], 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %s: %w", runConfigKeyKd, err)
	}

	tf, err := strconv.ParseInt(c[runConfigKeyTimeDividerNanoSec], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %s: %w", runConfigKeyTimeDividerNanoSec, err)
	}
	state.timeDivider = time.Duration(tf)

	state.countPolynomialCoefficients = make([]float64, 0)
	for _, v := range strings.Split(c[runConfigKeyActionCountPolynomialCoefficients], ",") {
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return nil, fmt.Errorf("unable to parse %s: %w", runConfigKeyActionCountPolynomialCoefficients, err)
		}
		state.countPolynomialCoefficients = append(state.countPolynomialCoefficients, f)
	}

	state.countQuantification = strings.ToLower(strings.TrimSpace(c[runConfigKeyActionCountQuantification]))
	if !utils.MatchAny([]string{state.countQuantification}, []string{"floor", "ceil", "ceiling", "round", "round_to_even"}) {
		return nil, fmt.Errorf("unable to parse %s: %w", runConfigKeyActionCountQuantification, err)
	}

	state.countMax, err = strconv.ParseFloat(c[runConfigKeyActionCountMax], 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %s: %w", runConfigKeyActionCountMax, err)
	}

	state.countMin, err = strconv.ParseFloat(c[runConfigKeyActionCountMin], 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %s: %w", runConfigKeyActionCountMin, err)
	}

	if state.countMax < state.countMin {
		return nil, fmt.Errorf("conflict: %s cannot be smaller than %s", runConfigKeyActionCountMax, runConfigKeyActionCountMin)
	}

	state.countDeadZone, err = strconv.ParseInt(c[runConfigKeyActionCountDeadZone], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %s: %w", runConfigKeyActionCountDeadZone, err)
	}

	s.states[id] = state
	return state, nil
}

func (s *StrategyPlugin) Run(eval *sdk.ScalingCheckEvaluation, count int64) (*sdk.ScalingCheckEvaluation, error) {
	// we cannot get any per-policy unique ID, so we try our best to create one
	id := fmt.Sprintf("%s/%d/%s/%s/%s", eval.Check.Source, utils.FNV64a(eval.Check.Query), eval.Check.Group, eval.Check.Name, eval.Check.Strategy.Name)
	s.logger.Debug("Run() called", "id", id)
	eval.Action.Direction = sdk.ScaleDirectionNone

	if len(eval.Metrics) == 0 {
		s.logger.Warn("Run() called with no data")

		// Notes:
		// The official examples return (nil, nil) here, but the plugin host would panic if you actually
		// return (nil, nil); and since nomad-autoscaler does not restart plugins for now, we can't recover
		// from it.
		// The workaround is to return a pseudo action.
		return eval, nil
	}

	state, err := s.newPolicy(id, eval.Check.Strategy.Config)
	if err != nil {
		return nil, fmt.Errorf("unable to parse strategy config: %w", err)
	}

	// inputs
	// Use only the latest value for now.
	measured := eval.Metrics[len(eval.Metrics)-1]

	// PID
	dt := float64(measured.Timestamp.Sub(state.previousTime) / state.timeDivider)
	proportional := state.target - measured.Value
	integral := state.integral + proportional*dt
	derivative := (proportional - state.previousError) / dt
	rawOutput := state.kp*proportional + state.ki*integral + state.kd*derivative

	// save internal state
	state.integral = integral
	state.previousError = proportional
	state.previousTime = measured.Timestamp
	// ignore the first sample
	if !state.hasPreviousData {
		s.logger.Info("first time here, not generating policies")
		state.hasPreviousData = true
		return eval, nil
	}

	// output transformation
	var tOutput float64
	// polynomial
	for p, k := range state.countPolynomialCoefficients {
		tOutput += k * math.Pow(rawOutput, float64(p))
	}
	// clamping
	tOutput = math.Min(tOutput, state.countMax)
	tOutput = math.Max(tOutput, state.countMin)
	// quantification
	var tOutputInt int64
	switch state.countQuantification {
	case "floor":
		tOutputInt = int64(math.Floor(tOutput))
	case "ceil", "ceiling":
		tOutputInt = int64(math.Ceil(tOutput))
	case "round":
		tOutputInt = int64(math.Round(tOutput))
	case "round_to_even":
		tOutputInt = int64(math.RoundToEven(tOutput))
	default:
		return nil, fmt.Errorf("unknown quantification method: %s", state.countQuantification)
	}
	// dead zone
	if utils.Abs(tOutputInt-count) <= utils.Abs(state.countDeadZone) {
		tOutputInt = count
	}

	eval.Action.Count = tOutputInt
	eval.Action.Reason = fmt.Sprintf("PID output: %f", rawOutput)
	if tOutputInt == count {
		eval.Action.Direction = sdk.ScaleDirectionNone
	} else if tOutputInt > count {
		eval.Action.Direction = sdk.ScaleDirectionUp
	} else {
		eval.Action.Direction = sdk.ScaleDirectionDown
	}

	s.logger.Trace("calculated scaling strategy results",
		"id", id,
		"metric_time", measured.Timestamp,
		"metric_value", measured.Value,
		"current_count", count,
		"raw_output", rawOutput,
		"new_count", tOutputInt,
		"direction", eval.Action.Direction,
	)

	return eval, nil
}
