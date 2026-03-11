package editorsystem

import (
	"testing"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	editoruicomponents "github.com/milk9111/sidescroller/cmd/editor/ui/components"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
)

func TestInspectorStateForSelectionReusesCachedState(t *testing.T) {
	system := &EditorUISystem{}
	catalog := &editorcomponent.PrefabCatalog{Items: []editorio.PrefabInfo{{
		Name:       "player",
		Path:       "player.yaml",
		EntityType: "player",
		Components: map[string]any{
			"transform": map[string]any{"x": 1.0, "y": 2.0},
		},
	}}}
	entities := &editorcomponent.LevelEntities{Items: []levels.Entity{{
		ID:   "player_1",
		Type: "player",
		X:    32,
		Y:    64,
		Props: map[string]interface{}{
			"prefab": "player.yaml",
		},
	}}}

	first := system.inspectorStateForSelection(catalog, entities, 0)
	if !first.Active {
		t.Fatal("expected inspector to be active for selected entity")
	}

	system.cachedInspectorState = editoruicomponents.InspectorState{Active: true, EntityLabel: "cached"}
	second := system.inspectorStateForSelection(catalog, entities, 0)
	if second.EntityLabel != "cached" {
		t.Fatalf("expected cached inspector state to be reused, got %q", second.EntityLabel)
	}
}

func TestInspectorStateForSelectionInvalidatesOnEntityChange(t *testing.T) {
	system := &EditorUISystem{}
	catalog := &editorcomponent.PrefabCatalog{Items: []editorio.PrefabInfo{{
		Name:       "player",
		Path:       "player.yaml",
		EntityType: "player",
		Components: map[string]any{
			"transform": map[string]any{"x": 1.0, "y": 2.0},
		},
	}}}
	entities := &editorcomponent.LevelEntities{Items: []levels.Entity{{
		ID:   "player_1",
		Type: "player",
		X:    32,
		Y:    64,
		Props: map[string]interface{}{
			"prefab": "player.yaml",
			"components": map[string]interface{}{
				"transform": map[string]interface{}{"x": 4.0},
			},
		},
	}}}

	first := system.inspectorStateForSelection(catalog, entities, 0)
	if got := inspectorFieldValue(first, "transform", "x"); got != "4" {
		t.Fatalf("expected initial transform.x to be 4, got %q", got)
	}

	entities.Items[0].Props["components"].(map[string]interface{})["transform"].(map[string]interface{})["x"] = 9.0
	second := system.inspectorStateForSelection(catalog, entities, 0)
	if got := inspectorFieldValue(second, "transform", "x"); got != "9" {
		t.Fatalf("expected updated transform.x to invalidate cache, got %q", got)
	}
}

func TestInspectorStateForSelectionReturnsHiddenDefaultsWithoutSelection(t *testing.T) {
	system := &EditorUISystem{}
	state := system.inspectorStateForSelection(nil, nil, -1)

	if state.Active {
		t.Fatal("expected inspector to be inactive without a selected entity")
	}
	if len(state.Sections) == 0 {
		t.Fatal("expected default inspector sections to be available without a selection")
	}
	if inspectorSectionVisible(state, "transform") {
		t.Fatal("expected transform section to remain hidden without a selection")
	}
	if inspectorFieldValue(state, "transform", "x") != "0" {
		t.Fatalf("expected default transform.x field to be constructed with zero value, got %q", inspectorFieldValue(state, "transform", "x"))
	}
}

func TestInspectorStateForSelectionShowsOnlyRelevantSections(t *testing.T) {
	system := &EditorUISystem{}
	catalog := &editorcomponent.PrefabCatalog{Items: []editorio.PrefabInfo{{
		Name:       "player",
		Path:       "player.yaml",
		EntityType: "player",
		Components: map[string]any{
			"transform": map[string]any{"x": 1.0, "y": 2.0},
		},
	}}}
	entities := &editorcomponent.LevelEntities{Items: []levels.Entity{{
		ID:   "player_1",
		Type: "player",
		X:    32,
		Y:    64,
		Props: map[string]interface{}{
			"prefab": "player.yaml",
		},
	}}}

	state := system.inspectorStateForSelection(catalog, entities, 0)
	if !state.Active {
		t.Fatal("expected inspector to be active for selected entity")
	}
	if !inspectorSectionVisible(state, "transform") {
		t.Fatal("expected transform section to be visible for selected entity")
	}
	if inspectorSectionVisible(state, "sprite") {
		t.Fatal("expected unrelated sprite section to stay hidden")
	}
	if got := inspectorFieldValue(state, "transform", "x"); got != "1" {
		t.Fatalf("expected transform.x to be populated from entity state, got %q", got)
	}
}

func TestCurrentTransitionSelectionIndexPrefersPendingTransition(t *testing.T) {
	selection := &editorcomponent.EntitySelectionState{SelectedIndex: 0}
	entities := &editorcomponent.LevelEntities{Items: []levels.Entity{{Type: "enemy"}, {Type: "transition"}}}
	pending := 1

	if got := currentTransitionSelectionIndex(selection, &pending, entities); got != 1 {
		t.Fatalf("expected pending transition selection 1, got %d", got)
	}
}

func TestMergeTransitionDraftKeepsDraftUntilWorldMatches(t *testing.T) {
	system := &EditorUISystem{transitionDraftSelection: 3, transitionDraft: &editoruicomponents.TransitionEditorState{
		Selected: true,
		ID:       "t1",
		ToLevel:  "zone_b.json",
		LinkedID: "upper_right",
		EnterDir: "left",
	}}
	base := editoruicomponents.TransitionEditorState{
		Selected: true,
		ID:       "t1",
		ToLevel:  "zone_b.json",
		LinkedID: "right",
		EnterDir: "down",
	}

	merged := system.mergeTransitionDraft(base, 3)
	if merged.LinkedID != "upper_right" {
		t.Fatalf("expected draft linked id to win, got %q", merged.LinkedID)
	}
	if merged.EnterDir != "left" {
		t.Fatalf("expected draft enter_dir to win, got %q", merged.EnterDir)
	}
	if system.transitionDraft == nil {
		t.Fatal("expected draft to remain until ECS matches it")
	}
}

func TestMergeTransitionDraftClearsWhenSelectionChanges(t *testing.T) {
	system := &EditorUISystem{transitionDraftSelection: 2, transitionDraft: &editoruicomponents.TransitionEditorState{Selected: true, LinkedID: "upper_right"}}
	base := editoruicomponents.TransitionEditorState{Selected: true, LinkedID: "right", EnterDir: "down"}

	merged := system.mergeTransitionDraft(base, 5)
	if merged.LinkedID != "right" {
		t.Fatalf("expected base state after selection change, got %q", merged.LinkedID)
	}
	if system.transitionDraft != nil {
		t.Fatal("expected draft to clear when selection changes")
	}
}

func TestTransitionEditDispatchStateWaitsForPendingSelection(t *testing.T) {
	system := &EditorUISystem{
		pendingTransitionEdit:          &editoruicomponents.TransitionEditorState{Selected: true, LinkedID: "upper_right"},
		pendingTransitionEditSelection: 2,
		pendingTransitionSelect:        intPtr(2),
	}
	selection := &editorcomponent.EntitySelectionState{SelectedIndex: -1}
	entities := &editorcomponent.LevelEntities{Items: []levels.Entity{{Type: "enemy"}, {Type: "enemy"}, {Type: "transition"}}}

	if got := system.transitionEditDispatchState(selection, entities); got != transitionEditWait {
		t.Fatalf("expected dispatch state wait, got %v", got)
	}
}

func TestTransitionEditDispatchStateReadyWhenSelectionMatches(t *testing.T) {
	system := &EditorUISystem{
		pendingTransitionEdit:          &editoruicomponents.TransitionEditorState{Selected: true, LinkedID: "upper_right"},
		pendingTransitionEditSelection: 1,
	}
	selection := &editorcomponent.EntitySelectionState{SelectedIndex: 1}
	entities := &editorcomponent.LevelEntities{Items: []levels.Entity{{Type: "enemy"}, {Type: "transition"}}}

	if got := system.transitionEditDispatchState(selection, entities); got != transitionEditReady {
		t.Fatalf("expected dispatch state ready, got %v", got)
	}
}

func TestTransitionEditDispatchStateStaleWhenSelectionMovedAway(t *testing.T) {
	system := &EditorUISystem{
		pendingTransitionEdit:          &editoruicomponents.TransitionEditorState{Selected: true, LinkedID: "upper_right"},
		pendingTransitionEditSelection: 1,
	}
	selection := &editorcomponent.EntitySelectionState{SelectedIndex: 0}
	entities := &editorcomponent.LevelEntities{Items: []levels.Entity{{Type: "transition"}, {Type: "transition"}}}

	if got := system.transitionEditDispatchState(selection, entities); got != transitionEditStale {
		t.Fatalf("expected dispatch state stale, got %v", got)
	}
}

func TestApplyTransitionEditorStateUpdatesLinkedID(t *testing.T) {
	item := &levels.Entity{Type: "transition", ID: "t1", Props: map[string]interface{}{"id": "t1", "linked_id": "right", "to_level": "zone_a.json", "enter_dir": "down"}}
	state := editoruicomponents.TransitionEditorState{Selected: true, ID: "t1", LinkedID: "upper_right", ToLevel: "zone_a", EnterDir: "left"}

	if !applyTransitionEditorState(item, state) {
		t.Fatal("expected transition state application to report changes")
	}
	if got := entityStringProp(*item, "linked_id"); got != "upper_right" {
		t.Fatalf("expected linked_id upper_right, got %q", got)
	}
	if got := entityStringProp(*item, "to_level"); got != "zone_a.json" {
		t.Fatalf("expected normalized to_level zone_a.json, got %q", got)
	}
	if got := entityStringProp(*item, "enter_dir"); got != "left" {
		t.Fatalf("expected enter_dir left, got %q", got)
	}
	if item.ID != "t1" {
		t.Fatalf("expected entity ID t1, got %q", item.ID)
	}
}

func TestFlushTransitionDraftToEntitiesAppliesSelectedDraft(t *testing.T) {
	w := ecs.NewWorld()
	system := &EditorUISystem{transitionDraftSelection: 0, transitionDraft: &editoruicomponents.TransitionEditorState{Selected: true, ID: "t1", LinkedID: "upper_right", ToLevel: "zone_b", EnterDir: "right"}}
	entities := &editorcomponent.LevelEntities{Items: []levels.Entity{{Type: "transition", ID: "t1", Props: map[string]interface{}{"id": "t1", "linked_id": "right", "to_level": "zone_a.json", "enter_dir": "down"}}}}
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{})

	if !system.flushTransitionDraftToEntities(w, entities) {
		t.Fatal("expected transition draft flush to apply changes")
	}
	if got := entityStringProp(entities.Items[0], "linked_id"); got != "upper_right" {
		t.Fatalf("expected linked_id upper_right after flush, got %q", got)
	}
}

func TestFlushTransitionDraftToEntitiesIgnoresCurrentSelectionLag(t *testing.T) {
	w := ecs.NewWorld()
	system := &EditorUISystem{transitionDraftSelection: 1, transitionDraft: &editoruicomponents.TransitionEditorState{Selected: true, ID: "t2", LinkedID: "upper_right", ToLevel: "zone_c", EnterDir: "up"}}
	entities := &editorcomponent.LevelEntities{Items: []levels.Entity{
		{Type: "enemy", ID: "enemy_1"},
		{Type: "transition", ID: "t2", Props: map[string]interface{}{"id": "t2", "linked_id": "right", "to_level": "zone_a.json", "enter_dir": "down"}},
	}}
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{})

	if !system.flushTransitionDraftToEntities(w, entities) {
		t.Fatal("expected transition draft flush to succeed even if ECS selection is lagging")
	}
	if got := entityStringProp(entities.Items[1], "linked_id"); got != "upper_right" {
		t.Fatalf("expected linked_id upper_right after lagged flush, got %q", got)
	}
	if got := entityStringProp(entities.Items[1], "to_level"); got != "zone_c.json" {
		t.Fatalf("expected normalized to_level zone_c.json, got %q", got)
	}
}

func TestSyncTransitionDraftToWorldAppliesImmediately(t *testing.T) {
	w := ecs.NewWorld()
	system := &EditorUISystem{
		transitionDraftSelection:       0,
		transitionDraft:                &editoruicomponents.TransitionEditorState{Selected: true, ID: "t1", LinkedID: "upper_right", ToLevel: "zone_b", EnterDir: "left"},
		pendingTransitionEditSelection: 0,
		pendingTransitionEdit:          &editoruicomponents.TransitionEditorState{Selected: true, ID: "t1", LinkedID: "upper_right", ToLevel: "zone_b", EnterDir: "left"},
	}
	selection := &editorcomponent.EntitySelectionState{SelectedIndex: 0}
	entities := &editorcomponent.LevelEntities{Items: []levels.Entity{{Type: "transition", ID: "t1", Props: map[string]interface{}{"id": "t1", "linked_id": "right", "to_level": "zone_a.json", "enter_dir": "down"}}}}
	session := &editorcomponent.EditorSession{}
	entity := ecs.CreateEntity(w)
	_ = ecs.Add(w, entity, editorcomponent.EditorSessionComponent.Kind(), session)
	_ = ecs.Add(w, entity, editorcomponent.EntitySelectionComponent.Kind(), selection)

	if !system.syncTransitionDraftToWorld(w, session, selection, entities) {
		t.Fatal("expected direct transition draft sync to apply changes")
	}
	if got := entityStringProp(entities.Items[0], "linked_id"); got != "upper_right" {
		t.Fatalf("expected linked_id upper_right, got %q", got)
	}
	if got := entityStringProp(entities.Items[0], "to_level"); got != "zone_b.json" {
		t.Fatalf("expected to_level zone_b.json, got %q", got)
	}
	if system.pendingTransitionEdit != nil {
		t.Fatal("expected pending transition edit to clear after direct sync")
	}
	if session.Status != "Updated transition properties" {
		t.Fatalf("expected updated status, got %q", session.Status)
	}
}

func intPtr(value int) *int {
	return &value
}

func inspectorFieldValue(state editoruicomponents.InspectorState, componentName, fieldName string) string {
	for _, section := range state.Sections {
		if section.Component != componentName {
			continue
		}
		for _, field := range section.Fields {
			if field.Field == fieldName {
				return field.Value
			}
		}
	}
	return ""
}

func inspectorSectionVisible(state editoruicomponents.InspectorState, componentName string) bool {
	for _, section := range state.Sections {
		if section.Component == componentName {
			return section.Visible
		}
	}
	return false
}
