package devbrowser

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/playwright-community/playwright-go"
)

type TargetSpec struct {
	Selector string
	AriaRole string
	AriaName string
	Nth      int
	Timeout  int
}

func (s TargetSpec) describe() string {
	parts := []string{}
	if strings.TrimSpace(s.Selector) != "" {
		parts = append(parts, fmt.Sprintf("selector=%q", s.Selector))
	}
	if strings.TrimSpace(s.AriaRole) != "" {
		parts = append(parts, fmt.Sprintf("aria_role=%q", s.AriaRole))
	}
	if strings.TrimSpace(s.AriaName) != "" {
		parts = append(parts, fmt.Sprintf("aria_name=%q", s.AriaName))
	}
	parts = append(parts, fmt.Sprintf("nth=%d", s.effectiveNth()))
	return strings.Join(parts, " ")
}

func (s TargetSpec) effectiveNth() int {
	if s.Nth < 1 {
		return 1
	}
	return s.Nth
}

func (s TargetSpec) timeoutMs() int {
	if s.Timeout < 1 {
		return 5_000
	}
	return s.Timeout
}

func resolveBounds(page playwright.Page, spec TargetSpec) (*playwright.Rect, error) {
	selector := strings.TrimSpace(spec.Selector)
	ariaRole := strings.TrimSpace(spec.AriaRole)
	ariaName := strings.TrimSpace(spec.AriaName)

	if selector == "" && ariaRole == "" {
		return nil, errors.New("selector or aria_role is required")
	}

	var locator playwright.Locator
	if selector != "" {
		locator = page.Locator(selector)
	} else {
		opts := playwright.PageGetByRoleOptions{}
		if ariaName != "" {
			opts.Name = ariaName
		}
		locator = page.GetByRole(playwright.AriaRole(ariaRole), opts)
	}

	target := locator
	nth := spec.effectiveNth()
	if nth > 1 {
		target = locator.Nth(nth - 1)
	}

	if err := target.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(float64(spec.timeoutMs())),
	}); err != nil {
		return nil, fmt.Errorf("target not found or not visible (%s): %w", spec.describe(), err)
	}

	box, err := target.BoundingBox()
	if err != nil {
		return nil, fmt.Errorf("failed to get bounds (%s): %w", spec.describe(), err)
	}
	if box == nil || box.Width <= 0 || box.Height <= 0 {
		return nil, fmt.Errorf("element has no bounding box (%s)", spec.describe())
	}
	return box, nil
}

func viewportSize(page playwright.Page) playwright.Size {
	if page == nil {
		def := DefaultWindowSize()
		return playwright.Size{Width: def.Width, Height: def.Height}
	}
	if vp := page.ViewportSize(); vp != nil {
		if vp.Width > 0 && vp.Height > 0 {
			return *vp
		}
	}
	def := DefaultWindowSize()
	return playwright.Size{Width: def.Width, Height: def.Height}
}

func clipWithPadding(box *playwright.Rect, padding int, viewport playwright.Size) (*playwright.Rect, error) {
	if padding < 0 {
		return nil, fmt.Errorf("padding_px must be non-negative")
	}
	vw := float64(viewport.Width)
	vh := float64(viewport.Height)
	if vw <= 0 || vh <= 0 {
		return nil, fmt.Errorf("viewport size is invalid")
	}

	x := box.X - float64(padding)
	y := box.Y - float64(padding)
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	width := box.Width + float64(padding*2)
	height := box.Height + float64(padding*2)

	maxWidth := vw - x
	maxHeight := vh - y
	width = math.Min(width, maxWidth)
	height = math.Min(height, maxHeight)

	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("computed clip is empty")
	}

	if width > 2000 {
		width = 2000
	}
	if height > 2000 {
		height = 2000
	}

	return &playwright.Rect{X: x, Y: y, Width: width, Height: height}, nil
}
