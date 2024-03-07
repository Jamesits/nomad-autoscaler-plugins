# strategy-pid

## Configuration

### Agent Configuration

```hcl
strategy "pid" {
  driver = "strategy-pid"
  args = [] # no args supported
  config = {} # see below
}
```

### Policy Configuration

```hcl
scaling "example" {
  # ...

  policy {
    check "example" {
      # ...

      strategy "pid" {
        target                = "0.0"
        proportional_factor   = "1.0"
        integral_factor       = "0.0"
        derivative_factor     = "0.0"
        time_divider_ns       = "1000000000"
        output_coefficients   = "0.0, 1.0"
        output_clamp_max      = "1000.0"
        output_clamp_min      = "0.0"
        output_quantification = "round"
        output_dead_zone      = "0"
      }
    }

    # ...
  }
}
```

PID algorithm arguments:

- `target`: float64, the target value of the controlled metric
- `proportional_factor`: float64, Kp
- `integral_factor`: float64, Ki
- `derivative_factor`: float64, Kd
- `time_divider_ns`: float64, dt = (current_eval_time - previous_eval_time) / time_divider_ns

PID output to strategy output signal path: polynomial -> clamp (min, max) -> quantification (float64 to int64) -> dead zone detection

Output transformation arguments:
- `output_coefficients`: comma-separated array of float64: Polynomial coefficients of the output transformation function
- `output_clamp_max`: float64, max value after the output transformation function
- `output_clamp_min`: float64, min value after the output transformation function
- `output_quantification`: string, the quantification method (`round`, `ceiling`, `floor`, `round_to_even`)
- `output_dead_zone`: int64, if abs(previous_output - current_output) <= abs(output_dead_zone), then don't bother do anything at all

Notes:
- The same arguments can be specified in either the policy configuration or the global plugin configuration
- Default values are shown in the example
- The quantification process may cause actual output value to be slightly out of bound by 1
- The time interval is not controlled by us and can have large jitters
- Please specify different *check* names globally; otherwise history data might screw up
