Outline shader (Kage)

This folder contains `outline.kage`, a small Kage fragment shader that draws a 1px outline
around opaque texels by sampling immediate neighbors and writing an `OutlineColor` in
transparent pixels adjacent to opaque ones.

Usage (Go / Ebiten):

1. Load the Kage source and create an `ebiten.Shader`. In some build setups you may compile the
   Kage source to platform shader bytecode; many workflows simply pass the source to `ebiten.NewShader`.

```go
// read shader source (adjust path as needed)
src, err := os.ReadFile("shaders/outline.kage")
if err != nil { return err }
shader, err := ebiten.NewShader(src)
if err != nil { return err }
```

2. When drawing a tile (image) with the shader, set the `Image0` image and uniforms:

```go
w := tileImg.Bounds().Dx()
h := tileImg.Bounds().Dy()
op := &ebiten.DrawRectShaderOptions{
    Images: map[string]*ebiten.Image{
        "Image0": tileImg,
    },
    Uniforms: map[string]interface{}{
        "TexelSize":    []float32{1.0 / float32(w), 1.0 / float32(h)},
        "Threshold":    float32(0.01),
        "OutlineColor": []float32{0.0, 0.0, 0.0, 1.0},
    },
}
// Draw onto the destination (screen or offscreen) at 1:1 pixel size
dst.DrawRectShader(float32(w), float32(h), shader, op)
```

Notes:
- `Threshold` controls alpha sensitivity when deciding whether a texel is "opaque".
- For thicker outlines sample diagonals or multiple texel offsets.
- If you want the outline behind the sprite, draw the shader result first, then draw the
  original sprite on top. Alternatively, render the shader to an offscreen and composite.

If you want, I can also add a small helper Go wrapper to centralize shader loading and drawing in
this repo (e.g. `cmd/editor/shader_helper.go`).