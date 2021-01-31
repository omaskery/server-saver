package jsonutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return fmt.Errorf("failed to decode duration JSON: %w", err)
	}

	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("failed to parse duration string: %w", err)
		}
		*d = Duration(tmp)
	default:
		return errors.New("invalid duration, must be integer number of nanoseconds or string duration")
	}

	return nil
}
