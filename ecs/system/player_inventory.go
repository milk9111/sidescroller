package system

import (
	"fmt"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/prefabs"
)

type inventoryItemDefinition struct {
	Prefab      string
	Name        string
	Description string
	FullText    string
	Range       float64
	Image       *ebiten.Image
}

var (
	inventoryItemDefinitionMu    sync.RWMutex
	inventoryItemDefinitionCache = map[string]*inventoryItemDefinition{}
)

func ensurePlayerInventory(w *ecs.World) *component.Inventory {
	if w == nil {
		return nil
	}

	if inventory := currentPlayerInventory(w); inventory != nil {
		return inventory
	}

	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return nil
	}

	inventory := &component.Inventory{}
	_ = ecs.Add(w, player, component.InventoryComponent.Kind(), inventory)
	return inventory
}

func currentPlayerInventory(w *ecs.World) *component.Inventory {
	if w == nil {
		return nil
	}

	if player, ok := ecs.First(w, component.PlayerTagComponent.Kind()); ok {
		if inventory, ok := ecs.Get(w, player, component.InventoryComponent.Kind()); ok && inventory != nil {
			return inventory
		}
	}

	if ent, ok := ecs.First(w, component.InventoryComponent.Kind()); ok {
		inventory, _ := ecs.Get(w, ent, component.InventoryComponent.Kind())
		return inventory
	}

	return nil
}

func addCollectedInventoryItem(w *ecs.World, e ecs.Entity, item *component.Item, sprite *component.Sprite, pickup *component.Pickup) {
	if w == nil || item == nil {
		item, _ = resolveEntityItem(w, e)
		if item == nil {
			return
		}
	}

	inventory := ensurePlayerInventory(w)
	if inventory == nil {
		return
	}

	prefabPath := inventoryPrefabPathForEntity(w, e, item)
	if prefabPath == "" {
		return
	}

	if item.Image == nil && sprite != nil {
		item.Image = sprite.Image
	}

	addInventoryItem(inventory, &component.InventoryItem{
		Prefab: prefabPath,
		Count:  1,
	})
}

func inventoryPrefabPathForEntity(w *ecs.World, e ecs.Entity, item *component.Item) string {
	if item != nil {
		if prefabPath := strings.TrimSpace(item.Prefab); prefabPath != "" {
			return prefabPath
		}
	}
	if w == nil || !e.Valid() {
		return ""
	}
	if itemReference, ok := ecs.Get(w, e, component.ItemReferenceComponent.Kind()); ok && itemReference != nil {
		return strings.TrimSpace(itemReference.Prefab)
	}
	return ""
}

func addInventoryItem(inventory *component.Inventory, item *component.InventoryItem) {
	if inventory == nil || item == nil {
		return
	}
	prefab := strings.TrimSpace(item.Prefab)
	if prefab == "" {
		return
	}
	count := max(1, item.Count)

	for index := range inventory.Items {
		if strings.TrimSpace(inventory.Items[index].Prefab) != prefab {
			continue
		}
		inventory.Items[index].Count += count
		return
	}

	inventory.Items = append(inventory.Items, component.InventoryItem{
		Prefab: prefab,
		Count:  count,
	})
}

func inventoryItemText(raw string) (string, string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "Item", ""
	}

	parts := strings.SplitN(trimmed, "\n", 2)
	name := strings.TrimSpace(parts[0])
	if len(parts) == 1 {
		return name, ""
	}
	return name, strings.TrimSpace(parts[1])
}

func resolveInventoryItemDefinition(prefabPath string) (*inventoryItemDefinition, error) {
	prefabPath = strings.TrimSpace(prefabPath)
	if prefabPath == "" {
		return nil, fmt.Errorf("inventory item prefab is empty")
	}

	inventoryItemDefinitionMu.RLock()
	if cached, ok := inventoryItemDefinitionCache[prefabPath]; ok && cached != nil {
		inventoryItemDefinitionMu.RUnlock()
		return cached, nil
	}
	inventoryItemDefinitionMu.RUnlock()

	spec, err := prefabs.LoadEntityBuildSpec(prefabPath)
	if err != nil {
		return nil, fmt.Errorf("load inventory item prefab %q: %w", prefabPath, err)
	}
	rawItem, ok := spec.Components["item"]
	if !ok {
		return nil, fmt.Errorf("inventory item prefab %q has no item component", prefabPath)
	}
	itemSpec, err := prefabs.DecodeComponentSpec[prefabs.ItemComponentSpec](rawItem)
	if err != nil {
		return nil, fmt.Errorf("decode inventory item prefab %q: %w", prefabPath, err)
	}

	imagePath := strings.TrimSpace(itemSpec.Image)
	if imagePath == "" {
		if rawSprite, ok := spec.Components["sprite"]; ok {
			spriteSpec, err := prefabs.DecodeComponentSpec[prefabs.SpriteComponentSpec](rawSprite)
			if err == nil {
				imagePath = strings.TrimSpace(spriteSpec.Image)
			}
		}
	}

	var image *ebiten.Image
	if imagePath != "" {
		loaded, err := assets.LoadImage(imagePath)
		if err != nil {
			return nil, fmt.Errorf("load inventory item image %q: %w", imagePath, err)
		}
		image = scaleInventoryImage(loaded, 4)
	}

	name, description := inventoryItemText(itemSpec.Description)
	definition := &inventoryItemDefinition{
		Prefab:      prefabPath,
		Name:        name,
		Description: description,
		FullText:    strings.TrimSpace(itemSpec.Description),
		Range:       itemSpec.Range,
		Image:       image,
	}

	inventoryItemDefinitionMu.Lock()
	inventoryItemDefinitionCache[prefabPath] = definition
	inventoryItemDefinitionMu.Unlock()
	return definition, nil
}

func resolveEntityItem(w *ecs.World, e ecs.Entity) (*component.Item, *component.Sprite) {
	if w == nil || !e.Valid() || !ecs.IsAlive(w, e) {
		return nil, nil
	}

	sprite, _ := ecs.Get(w, e, component.SpriteComponent.Kind())
	if item, ok := ecs.Get(w, e, component.ItemComponent.Kind()); ok && item != nil {
		return item, sprite
	}

	itemReference, ok := ecs.Get(w, e, component.ItemReferenceComponent.Kind())
	if !ok || itemReference == nil {
		return nil, sprite
	}

	definition, err := resolveInventoryItemDefinition(itemReference.Prefab)
	if err != nil || definition == nil {
		return nil, sprite
	}

	return &component.Item{
		Prefab:      definition.Prefab,
		Description: definition.FullText,
		Range:       definition.Range,
		Image:       definition.Image,
	}, sprite
}

func scaleInventoryImage(src *ebiten.Image, factor float64) *ebiten.Image {
	if src == nil || factor <= 0 || factor == 1 {
		return src
	}
	bounds := src.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return src
	}
	dst := ebiten.NewImage(int(float64(bounds.Dx())*factor), int(float64(bounds.Dy())*factor))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(factor, factor)
	dst.DrawImage(src, op)
	return dst
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
