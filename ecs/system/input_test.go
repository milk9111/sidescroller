package system

import "testing"

func TestShouldTriggerUpwardAttack(t *testing.T) {
	tests := []struct {
		name          string
		attackPressed bool
		aimY          float64
		keyboardUp    bool
		usingGamepad  bool
		want          bool
	}{
		{name: "requires attack press", attackPressed: false, aimY: -1, keyboardUp: true, usingGamepad: true, want: false},
		{name: "keyboard up triggers upward attack", attackPressed: true, aimY: 0, keyboardUp: true, usingGamepad: false, want: true},
		{name: "keyboard up still triggers upward attack when gamepad is connected", attackPressed: true, aimY: 0, keyboardUp: true, usingGamepad: true, want: true},
		{name: "keyboard mouse aim up still triggers upward attack", attackPressed: true, aimY: -0.1, usingGamepad: false, want: true},
		{name: "keyboard neutral input stays regular attack", attackPressed: true, aimY: 0, keyboardUp: false, usingGamepad: false, want: false},
		{name: "gamepad slight upward tilt stays regular attack", attackPressed: true, aimY: -0.25, keyboardUp: false, usingGamepad: true, want: false},
		{name: "gamepad medium upward tilt still stays regular attack", attackPressed: true, aimY: -0.59, keyboardUp: false, usingGamepad: true, want: false},
		{name: "gamepad strong upward tilt triggers upward attack", attackPressed: true, aimY: -0.6, keyboardUp: false, usingGamepad: true, want: true},
		{name: "gamepad full upward tilt triggers upward attack", attackPressed: true, aimY: -1, keyboardUp: false, usingGamepad: true, want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := shouldTriggerUpwardAttack(test.attackPressed, test.aimY, test.keyboardUp, test.usingGamepad)
			if got != test.want {
				t.Fatalf("shouldTriggerUpwardAttack(%t, %v, %t, %t) = %t, want %t", test.attackPressed, test.aimY, test.keyboardUp, test.usingGamepad, got, test.want)
			}
		})
	}
}
