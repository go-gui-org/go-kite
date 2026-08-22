package main

import (
	"math"
	"testing"
)

func TestScaledImageDimsDefaultsDegenerateInputs(t *testing.T) {
	w, h := scaledImageDims(0, 0)
	if w != imageWidth || h != maxImageHeight {
		t.Fatalf("zero dims: got %vx%v, want %vx%v", w, h, imageWidth, maxImageHeight)
	}
	w, h = scaledImageDims(-10, -5)
	if w != imageWidth || h != maxImageHeight {
		t.Fatalf("negative dims: got %vx%v, want %vx%v", w, h, imageWidth, maxImageHeight)
	}
	w, h = scaledImageDims(500, 0)
	if w != imageWidth || h != maxImageHeight {
		t.Fatalf("zero height: got %vx%v, want %vx%v", w, h, imageWidth, maxImageHeight)
	}
	w, h = scaledImageDims(0, 500)
	if w != imageWidth || h != maxImageHeight {
		t.Fatalf("zero width: got %vx%v, want %vx%v", w, h, imageWidth, maxImageHeight)
	}
}

func TestScaledImageDimsScalesWideImages(t *testing.T) {
	// 640x480 → 270 wide, height kept in proportion (480*270/640).
	w, h := scaledImageDims(640, 480)
	if w != imageWidth {
		t.Fatalf("width: got %v, want %v", w, imageWidth)
	}
	if want := float32(480) * (float32(imageWidth) / 640); h != want {
		t.Fatalf("height: got %v, want %v", h, want)
	}
}

func TestScaledImageDimsCapsHeight(t *testing.T) {
	// Portrait embed taller than the cap, both before and after scaling.
	if _, h := scaledImageDims(imageWidth, 800); h != maxImageHeight {
		t.Fatalf("height: got %v, want %v", h, maxImageHeight)
	}
	if _, h := scaledImageDims(640, 3000); h != maxImageHeight {
		t.Fatalf("scaled height: got %v, want %v", h, maxImageHeight)
	}
}

func TestScaledImageDimsFloorsSubPixelHeight(t *testing.T) {
	// Absurdly wide embed scales height below 1px; floor keeps it visible.
	w, h := scaledImageDims(1e9, 10)
	if w != imageWidth {
		t.Fatalf("width: got %v, want %v", w, imageWidth)
	}
	if h != 1 {
		t.Fatalf("height: got %v, want 1", h)
	}
}

func TestScaledImageDimsRejectsNaNAndInf(t *testing.T) {
	// Every comparison against NaN is false, so without the up-front
	// guard a NaN dim sails through all four branches unchanged and
	// poisons the layout with NaN extents.
	w, h := scaledImageDims(float32(math.NaN()), float32(math.NaN()))
	if w != imageWidth || h != maxImageHeight {
		t.Fatalf("NaN dims: got %vx%v, want %vx%v", w, h, imageWidth, maxImageHeight)
	}
	w, h = scaledImageDims(float32(math.NaN()), 100)
	if w != imageWidth || h != 100 {
		t.Fatalf("NaN width: got %vx%v, want %vx%v", w, h, imageWidth, 100)
	}
	// +Inf width scales the height to zero, which then falls into the
	// display default rather than returning a vanishing embed.
	w, h = scaledImageDims(float32(math.Inf(1)), 480)
	if w != imageWidth || h != maxImageHeight {
		t.Fatalf("+Inf width: got %vx%v, want %vx%v", w, h, imageWidth, maxImageHeight)
	}
	_, h = scaledImageDims(640, float32(math.Inf(1)))
	if h != maxImageHeight {
		t.Fatalf("+Inf height: got %v, want %v", h, maxImageHeight)
	}
	// -Inf is caught by the plain <= 0 branches.
	w, h = scaledImageDims(float32(math.Inf(-1)), float32(math.Inf(-1)))
	if w != imageWidth || h != maxImageHeight {
		t.Fatalf("-Inf dims: got %vx%v, want %vx%v", w, h, imageWidth, maxImageHeight)
	}
}
