package ecs

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs/component"
)

func TestSparseWorldEntityLifecycle(t *testing.T) {
	cases := []struct {
		name         string
		create       int
		destroyIndex int // -1 = none
	}{
		{"single", 1, 0},
		{"three_create_destroy_middle", 3, 1},
		{"none_destroy", 2, -1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := NewWorld()
			ents := make([]Entity, 0, c.create)
			for i := 0; i < c.create; i++ {
				ents = append(ents, CreateEntity(w))
			}
			if len(Entities(w)) != c.create {
				t.Fatalf("expected %d entities, got %d", c.create, len(Entities(w)))
			}
			if c.destroyIndex >= 0 {
				if !DestroyEntity(w, ents[c.destroyIndex]) {
					t.Fatalf("DestroyEntity should return true for alive entity")
				}
				if IsAlive(w, ents[c.destroyIndex]) {
					t.Fatalf("entity should not be alive after destruction")
				}
			}
		})
	}
}

func toSet(ents []Entity) map[Entity]struct{} {
	m := make(map[Entity]struct{}, len(ents))
	for _, e := range ents {
		m[e] = struct{}{}
	}
	return m
}

func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}

func TestSparseWorldComponentsAndQueries(t *testing.T) {
	t.Run("component_table", func(t *testing.T) {
		w := NewWorld()

		h1 := component.NewComponent[int]()
		h2 := component.NewComponent[string]()
		h3 := component.NewComponent[float64]()

		e1 := CreateEntity(w)
		e2 := CreateEntity(w)

		tests := []struct {
			name     string
			setup    func() error
			check    func(t *testing.T)
			teardown func() bool
		}{
			{
				name:  "add_int_to_e1",
				setup: func() error { return Add(w, e1, h1.Kind(), intPtr(10)) },
				check: func(t *testing.T) {
					v, ok := Get[int](w, e1, h1.Kind())
					if !ok || *v != 10 {
						t.Fatalf("expected 10, got %v ok=%v", v, ok)
					}
				},
				teardown: func() bool { return Remove[int](w, e1, h1.Kind()) },
			},
			{
				name: "add_str_to_e1_and_e2",
				setup: func() error {
					if err := Add(w, e1, h2.Kind(), stringPtr("a")); err != nil {
						return err
					}
					return Add(w, e2, h2.Kind(), stringPtr("b"))
				},
				check: func(t *testing.T) {
					if !Has[string](w, e1, h2.Kind()) || !Has[string](w, e2, h2.Kind()) {
						t.Fatalf("expected both entities to have string component")
					}
				},
				teardown: func() bool { return Remove[string](w, e1, h2.Kind()) },
			},
			{
				name:  "add_float_and_remove",
				setup: func() error { return Add(w, e1, h3.Kind(), float64Ptr(1.23)) },
				check: func(t *testing.T) {
					if _, ok := Get[float64](w, e1, h3.Kind()); !ok {
						t.Fatalf("expected float present")
					}
				},
				teardown: func() bool { return Remove[float64](w, e1, h3.Kind()) },
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				if err := tc.setup(); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				tc.check(t)
				if !tc.teardown() {
					t.Fatalf("teardown failed for %s", tc.name)
				}
			})
		}
	})
}

func TestForEach(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		w := NewWorld()
		h := component.NewComponent[int]()

		e1 := CreateEntity(w)
		e2 := CreateEntity(w)
		e3 := CreateEntity(w)

		if err := Add(w, e1, h.Kind(), intPtr(1)); err != nil {
			t.Fatalf("add failed: %v", err)
		}
		if err := Add(w, e3, h.Kind(), intPtr(3)); err != nil {
			t.Fatalf("add failed: %v", err)
		}

		var ents []Entity
		ForEach(w, h.Kind(), func(e Entity, _ *int) { ents = append(ents, e) })
		set := toSet(ents)

		if _, ok := set[e1]; !ok {
			t.Fatalf("expected e1 in ForEach result")
		}
		if _, ok := set[e3]; !ok {
			t.Fatalf("expected e3 in ForEach result")
		}
		if _, ok := set[e2]; ok {
			t.Fatalf("did not expect e2 in ForEach result")
		}
	})
}

func TestForEach3(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "intersection",
			run: func(t *testing.T) {
				w := NewWorld()
				e1 := CreateEntity(w)
				e2 := CreateEntity(w)
				e3 := CreateEntity(w)
				e4 := CreateEntity(w)

				ka := component.NewComponentKind[int]()
				kb := component.NewComponentKind[int]()
				kc := component.NewComponentKind[int]()

				if err := Add(w, e1, ka, intPtr(1)); err != nil {
					t.Fatal(err)
				}
				if err := Add(w, e2, ka, intPtr(2)); err != nil {
					t.Fatal(err)
				}
				if err := Add(w, e2, kb, intPtr(3)); err != nil {
					t.Fatal(err)
				}
				if err := Add(w, e2, kc, intPtr(5)); err != nil {
					t.Fatal(err)
				}
				if err := Add(w, e3, kb, intPtr(4)); err != nil {
					t.Fatal(err)
				}
				if err := Add(w, e4, kc, intPtr(6)); err != nil {
					t.Fatal(err)
				}

				var res []Entity
				ForEach3(w, ka, kb, kc, func(e Entity, _ *int, _ *int, _ *int) { res = append(res, e) })
				if len(res) != 1 || res[0].id() != e2.id() {
					t.Fatalf("expected only e2, got %v", res)
				}
			},
		},
		{
			name: "ignores_dead_entities",
			run: func(t *testing.T) {
				w := NewWorld()
				e := CreateEntity(w)

				ka := component.NewComponentKind[int]()
				kb := component.NewComponentKind[int]()
				kc := component.NewComponentKind[int]()

				if err := Add(w, e, ka, intPtr(1)); err != nil {
					t.Fatal(err)
				}
				if err := Add(w, e, kb, intPtr(2)); err != nil {
					t.Fatal(err)
				}
				if err := Add(w, e, kc, intPtr(3)); err != nil {
					t.Fatal(err)
				}

				if !DestroyEntity(w, e) {
					t.Fatal("failed to destroy entity")
				}

				var res []Entity
				ForEach3(w, ka, kb, kc, func(e Entity, _ *int, _ *int, _ *int) { res = append(res, e) })
				if len(res) != 0 {
					t.Fatalf("expected empty result after destroy, got %v", res)
				}
			},
		},
		{
			name: "no_common",
			run: func(t *testing.T) {
				w := NewWorld()
				e1 := CreateEntity(w)
				e2 := CreateEntity(w)

				ka := component.NewComponentKind[int]()
				kb := component.NewComponentKind[int]()
				kc := component.NewComponentKind[int]()

				if err := Add(w, e1, ka, intPtr(1)); err != nil {
					t.Fatal(err)
				}
				if err := Add(w, e2, kb, intPtr(2)); err != nil {
					t.Fatal(err)
				}

				var res []Entity
				ForEach3(w, ka, kb, kc, func(e Entity, _ *int, _ *int, _ *int) { res = append(res, e) })
				if len(res) != 0 {
					t.Fatalf("expected no common entities, got %v", res)
				}
			},
		},
		{
			name: "missing_store_returns_nil",
			run: func(t *testing.T) {
				w := NewWorld()
				e := CreateEntity(w)

				ka := component.NewComponentKind[int]()
				kb := component.NewComponentKind[int]()
				kc := component.NewComponentKind[int]()

				if err := Add(w, e, ka, intPtr(1)); err != nil {
					t.Fatal(err)
				}

				var res []Entity
				ForEach3(w, ka, kb, kc, func(e Entity, _ *int, _ *int, _ *int) { res = append(res, e) })
				if res != nil && len(res) != 0 {
					t.Fatalf("expected empty when other store missing, got %v", res)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}

func TestForEach4(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "intersection",
			run: func(t *testing.T) {
				w := NewWorld()
				e1 := CreateEntity(w)
				e2 := CreateEntity(w)
				e3 := CreateEntity(w)
				e4 := CreateEntity(w)
				e5 := CreateEntity(w)

				ka := component.NewComponentKind[int]()
				kb := component.NewComponentKind[int]()
				kc := component.NewComponentKind[int]()
				kd := component.NewComponentKind[int]()

				// e2 will be the only entity having all four
				if err := Add(w, e1, ka, intPtr(1)); err != nil {
					t.Fatal(err)
				}
				if err := Add(w, e2, ka, intPtr(2)); err != nil {
					t.Fatal(err)
				}
				if err := Add(w, e2, kb, intPtr(3)); err != nil {
					t.Fatal(err)
				}
				if err := Add(w, e2, kc, intPtr(5)); err != nil {
					t.Fatal(err)
				}
				if err := Add(w, e2, kd, intPtr(7)); err != nil {
					t.Fatal(err)
				}
				if err := Add(w, e3, kb, intPtr(4)); err != nil {
					t.Fatal(err)
				}
				if err := Add(w, e4, kc, intPtr(6)); err != nil {
					t.Fatal(err)
				}
				if err := Add(w, e5, kd, intPtr(8)); err != nil {
					t.Fatal(err)
				}

				var res []Entity
				ForEach4(w, ka, kb, kc, kd, func(e Entity, _ *int, _ *int, _ *int, _ *int) { res = append(res, e) })
				if len(res) != 1 || res[0].id() != e2.id() {
					t.Fatalf("expected only e2, got %v", res)
				}
			},
		},
		{
			name: "ignores_dead_entities",
			run: func(t *testing.T) {
				w := NewWorld()
				e := CreateEntity(w)

				ka := component.NewComponentKind[int]()
				kb := component.NewComponentKind[int]()
				kc := component.NewComponentKind[int]()
				kd := component.NewComponentKind[int]()

				if err := Add(w, e, ka, intPtr(1)); err != nil {
					t.Fatal(err)
				}
				if err := Add(w, e, kb, intPtr(2)); err != nil {
					t.Fatal(err)
				}
				if err := Add(w, e, kc, intPtr(3)); err != nil {
					t.Fatal(err)
				}
				if err := Add(w, e, kd, intPtr(4)); err != nil {
					t.Fatal(err)
				}

				if !DestroyEntity(w, e) {
					t.Fatal("failed to destroy entity")
				}

				var res []Entity
				ForEach4(w, ka, kb, kc, kd, func(e Entity, _ *int, _ *int, _ *int, _ *int) { res = append(res, e) })
				if len(res) != 0 {
					t.Fatalf("expected empty result after destroy, got %v", res)
				}
			},
		},
		{
			name: "no_common",
			run: func(t *testing.T) {
				w := NewWorld()
				e1 := CreateEntity(w)
				e2 := CreateEntity(w)

				ka := component.NewComponentKind[int]()
				kb := component.NewComponentKind[int]()
				kc := component.NewComponentKind[int]()
				kd := component.NewComponentKind[int]()

				if err := Add(w, e1, ka, intPtr(1)); err != nil {
					t.Fatal(err)
				}
				if err := Add(w, e2, kb, intPtr(2)); err != nil {
					t.Fatal(err)
				}

				var res []Entity
				ForEach4(w, ka, kb, kc, kd, func(e Entity, _ *int, _ *int, _ *int, _ *int) { res = append(res, e) })
				if len(res) != 0 {
					t.Fatalf("expected no common entities, got %v", res)
				}
			},
		},
		{
			name: "missing_store_returns_nil",
			run: func(t *testing.T) {
				w := NewWorld()
				e := CreateEntity(w)

				ka := component.NewComponentKind[int]()
				kb := component.NewComponentKind[int]()
				kc := component.NewComponentKind[int]()
				kd := component.NewComponentKind[int]()

				if err := Add(w, e, ka, intPtr(1)); err != nil {
					t.Fatal(err)
				}

				var res []Entity
				ForEach4(w, ka, kb, kc, kd, func(e Entity, _ *int, _ *int, _ *int, _ *int) { res = append(res, e) })
				if res != nil && len(res) != 0 {
					t.Fatalf("expected empty when other store missing, got %v", res)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}
