package system

import "testing"

func TestShouldTriggerUpwardAttack(t *testing.T) {
	tests := []struct {
		name         string
		attackPressed bool
		aimY         float64
		usingGamepad bool
		want         bool
	}{
		{name: "requires attack press", attackPressed: false, aimY: -1, usingGamepad: true, want: false},
		{name: "keyboard upward attack keeps current behavior", attackPressed: true, aimY: -0.1, usingGamepad: false, want: true},
		{name: "gamepad slight upward tilt stays regular attack", attackPressed: true, aimY: -0.25, usingGamepad: true, want: false},
		{name: "gamepad medium upward tilt still stays regular attack", attackPressed: true, aimY: -0.59, usingGamepad: true, want: false},
		{name: "gamepad strong upward tilt triggers upward attack", attackPressed: true, aimY: -0.6, usingGamepad: true, want: true},
		{name: "gamepad full upward tilt triggers upward attack", attackPressed: true, aimY: -1, usingGamepad: true, want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := shouldTriggerUpwardAttack(test.attackPressed, test.aimY, test.usingGamepad)
			if got != test.want {
				t.Fatalf("shouldTriggerUpwardAttack(%t, %v, %t) = %t, want %t", test.attackPressed, test.aimY, test.usingGamepad, got, test.want)
			}
		})
	}
}
