package editorsystem

import (
	"strings"
	"testing"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	editorui "github.com/milk9111/sidescroller/cmd/editor/ui"
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
	if got := first.DocumentText; !strings.Contains(got, "x: 4") {
		t.Fatalf("expected initial document to contain override x: 4, got %q", got)
	}

	entities.Items[0].Props["components"].(map[string]interface{})["transform"].(map[string]interface{})["x"] = 9.0
	second := system.inspectorStateForSelection(catalog, entities, 0)
	if got := second.DocumentText; !strings.Contains(got, "x: 9") {
		t.Fatalf("expected updated document to invalidate cache, got %q", got)
	}
}

func TestInspectorStateForSelectionReturnsHiddenDefaultsWithoutSelection(t *testing.T) {
	system := &EditorUISystem{}
	state := system.inspectorStateForSelection(nil, nil, -1)

	if state.Active {
		t.Fatal("expected inspector to be inactive without a selected entity")
	}
	if state.DocumentText != "" {
		t.Fatalf("expected no document without selection, got %q", state.DocumentText)
	}
	if state.StatusMessage != "Select an entity to inspect" {
		t.Fatalf("expected empty-selection status message, got %q", state.StatusMessage)
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
	if !strings.Contains(state.DocumentText, "transform:") {
		t.Fatalf("expected transform component in document, got %q", state.DocumentText)
	}
	if strings.Contains(state.DocumentText, "sprite:") {
		t.Fatalf("expected unrelated sprite component to stay absent, got %q", state.DocumentText)
	}
	if !strings.Contains(state.DocumentText, "x: 1") {
		t.Fatalf("expected transform.x to be populated from entity state, got %q", state.DocumentText)
	}
}

func TestInspectorStateForSelectionUsesEntityOverridesWithoutPrefab(t *testing.T) {
	system := &EditorUISystem{}
	entities := &editorcomponent.LevelEntities{Items: []levels.Entity{{
		ID:   "orphan_1",
		Type: "orphan",
		Props: map[string]interface{}{
			"components": map[string]interface{}{
				"script": map[string]interface{}{"name": "custom"},
			},
		},
	}}}

	state := system.inspectorStateForSelection(nil, entities, 0)
	if !state.Active {
		t.Fatal("expected inspector to be active for selected entity")
	}
	if state.PrefabPath != "" {
		t.Fatalf("expected no prefab path for entity without prefab, got %q", state.PrefabPath)
	}
	if !strings.Contains(state.DocumentText, "script:") || !strings.Contains(state.DocumentText, "name: custom") {
		t.Fatalf("expected entity-local overrides to populate document, got %q", state.DocumentText)
	}
}

func TestBuildInspectorDocumentSortsTopLevelComponents(t *testing.T) {
	document, err := buildInspectorDocument(map[string]any{
		"sprite":    map[string]any{"image": "player.png"},
		"transform": map[string]any{"x": 1, "y": 2},
	})
	if err != nil {
		t.Fatalf("buildInspectorDocument() error = %v", err)
	}
	transformIndex := strings.Index(document, "transform:")
	spriteIndex := strings.Index(document, "sprite:")
	if transformIndex < 0 || spriteIndex < 0 {
		t.Fatalf("expected both components in document, got %q", document)
	}
	if transformIndex > spriteIndex {
		t.Fatalf("expected transform to be emitted before sprite, got %q", document)
	}
}

func TestEditorUISystemRoutesPendingInspectorDocumentToAction(t *testing.T) {
	system := &EditorUISystem{pendingInspectorDocument: stringPtr("transform:\n  x: 4")}
	actions := &editorcomponent.EditorActions{SelectLayer: -1, SelectEntity: -1}

	system.dispatchPendingInspectorDocument(actions, 0)

	if !actions.ApplyInspectorDocument {
		t.Fatal("expected inspector document apply action to be set")
	}
	if got := actions.InspectorDocument; got != "transform:\n  x: 4" {
		t.Fatalf("expected inspector document to be forwarded, got %q", got)
	}
	if system.pendingInspectorDocument != nil {
		t.Fatal("expected pending inspector document to be cleared after dispatch")
	}
	if actions.ApplyTransitionFields || actions.ApplyGateFields {
		t.Fatal("expected only the inspector document action to be touched")
	}
}

func TestEditorUISystemInspectorDocumentDispatchIsNoopWithoutPendingDocument(t *testing.T) {
	system := &EditorUISystem{}
	actions := &editorcomponent.EditorActions{SelectLayer: -1, SelectEntity: -1}

	system.dispatchPendingInspectorDocument(actions, -1)

	if actions.ApplyInspectorDocument {
		t.Fatal("expected no inspector document action without pending document")
	}
	if actions.InspectorDocument != "" {
		t.Fatalf("expected no inspector document payload, got %q", actions.InspectorDocument)
	}
}

func TestEditorUISystemDispatchPendingInspectorDocumentTracksPendingFeedback(t *testing.T) {
	system := &EditorUISystem{pendingInspectorDocument: stringPtr("transform:\n  x: 4"), inspectorFeedbackSelection: -1}
	actions := &editorcomponent.EditorActions{SelectLayer: -1, SelectEntity: -1}

	system.dispatchPendingInspectorDocument(actions, 3)

	if !system.inspectorApplyPending {
		t.Fatal("expected inspector feedback to wait for apply result")
	}
	if system.inspectorFeedbackSelection != 3 {
		t.Fatalf("expected inspector feedback selection 3, got %d", system.inspectorFeedbackSelection)
	}
	if system.inspectorFeedbackParseError != "" || system.inspectorFeedbackStatus != "" {
		t.Fatalf("expected pending feedback to clear stale messages, got status=%q parse=%q", system.inspectorFeedbackStatus, system.inspectorFeedbackParseError)
	}
}

func TestEditorUISystemSyncInspectorFeedbackCapturesParseFailure(t *testing.T) {
	system := &EditorUISystem{inspectorApplyPending: true, inspectorFeedbackSelection: 2}
	session := &editorcomponent.EditorSession{Status: "Inspector apply failed: yaml: line 3: did not find expected key"}

	system.syncInspectorFeedback(session, 2)

	if system.inspectorApplyPending {
		t.Fatal("expected parse failure to resolve the pending inspector apply")
	}
	if system.inspectorFeedbackStatus != "Inspector apply failed" {
		t.Fatalf("expected failure status label, got %q", system.inspectorFeedbackStatus)
	}
	if system.inspectorFeedbackParseError != "yaml: line 3: did not find expected key" {
		t.Fatalf("expected parse error message to be captured, got %q", system.inspectorFeedbackParseError)
	}

	system.syncInspectorFeedback(session, 5)
	if system.inspectorFeedbackSelection != -1 || system.inspectorFeedbackStatus != "" || system.inspectorFeedbackParseError != "" {
		t.Fatalf("expected selection change to clear stale inspector feedback, got %+v", system)
	}
}

func TestEditorUISystemDecorateInspectorStateKeepsDirtyOnParseFailure(t *testing.T) {
	theme, err := editoruicomponents.NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	panel := editoruicomponents.NewInspectorPanel(theme, nil)
	panel.Editor.SetText("transform:\n  x: broken")
	panel.Editor.SetDirty(true)
	system := &EditorUISystem{
		ui:                          &editorui.EditorUI{AssetPanel: &editoruicomponents.AssetPanel{Inspector: panel}},
		inspectorFeedbackSelection:  1,
		inspectorFeedbackStatus:     "Inspector apply failed",
		inspectorFeedbackParseError: "yaml: line 2: did not find expected key",
	}

	state := system.decorateInspectorState(editoruicomponents.InspectorState{Active: true, EntityLabel: "enemy", DocumentText: "transform:\n  x: 32"}, 1)

	if !state.Dirty {
		t.Fatal("expected parse failure to keep the editor dirty")
	}
	if state.ParseError != "yaml: line 2: did not find expected key" {
		t.Fatalf("expected parse error to surface in inspector state, got %q", state.ParseError)
	}
	if state.StatusMessage != "Inspector apply failed" {
		t.Fatalf("expected failure status message, got %q", state.StatusMessage)
	}
}

func TestEditorUISystemDecorateInspectorStateClearsDirtyAfterSuccessfulApply(t *testing.T) {
	theme, err := editoruicomponents.NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	panel := editoruicomponents.NewInspectorPanel(theme, nil)
	panel.Editor.SetText("transform:\n  x: 4")
	panel.Editor.SetDirty(true)
	system := &EditorUISystem{
		ui:                         &editorui.EditorUI{AssetPanel: &editoruicomponents.AssetPanel{Inspector: panel}},
		inspectorFeedbackSelection: 1,
		inspectorFeedbackStatus:    "Updated entity component overrides",
	}

	state := system.decorateInspectorState(editoruicomponents.InspectorState{Active: true, EntityLabel: "enemy", DocumentText: "transform:\n  x: 4"}, 1)

	if state.Dirty {
		t.Fatal("expected successful apply to clear the inspector dirty flag")
	}
	if state.StatusMessage != "Updated entity component overrides" {
		t.Fatalf("expected success status message, got %q", state.StatusMessage)
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

func stringPtr(value string) *string {
	return &value
}
