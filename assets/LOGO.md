# Mycelium Logo Usage Guide

## Logo Files

The Mycelium project uses adaptive logos that automatically switch based on the viewer's GitHub theme preference.

### Required Files

Place these files in the `assets/` directory:

1. **`logo-light.png`** - Logo for light mode (dark text/graphics)
2. **`logo-dark.png`** - Logo for dark mode (light text/graphics)

### Logo Design

The Mycelium logo features three glowing mushrooms representing the three tools:
- 🔍 **Truffle** (left) - Find and discover resources
- 🚀 **Spawn** (center) - Launch instances
- 🤖 **Spored** (right) - Monitor and manage

Connected by an underground mycelium network, representing the interconnected nature of the toolset.

## Usage in Documentation

### GitHub Adaptive Theme Syntax

Use this syntax in markdown files to automatically show the correct logo based on user's theme:

```markdown
<p align="center">
  <img src="assets/logo-light.png#gh-light-mode-only" alt="Mycelium Logo" width="600">
  <img src="assets/logo-dark.png#gh-dark-mode-only" alt="Mycelium Logo" width="600">
</p>
```

### How It Works

GitHub recognizes the URL fragments:
- `#gh-light-mode-only` - Shows only in light mode
- `#gh-dark-mode-only` - Shows only in dark mode

The browser automatically hides the inappropriate image based on the user's `prefers-color-scheme` setting.

### Recommended Widths

- **Main README**: 600px - Large and prominent
- **Component READMEs**: 500px - Slightly smaller but still visible
- **Documentation pages**: 400px - Compact for embedded content

## Integration Status

### ✅ Integrated

- [x] Main README (`README.md`)
- [x] Spawn README (`spawn/README.md`)
- [x] Truffle README (`truffle/README.md`)

### 📋 Future Integration

- [ ] GitHub repository social preview (`.github/preview.png`)
- [ ] GitHub repository description image
- [ ] Documentation site favicon
- [ ] CLI `--help` banner (ASCII art version)

## GitHub Repository Settings

### Social Preview Image

To set the repository social preview image:

1. Go to repository Settings
2. Scroll to "Social preview"
3. Upload a 1280x640px image
4. Recommendation: Use the dark logo version as it works better on most social platforms

### Repository Image

GitHub shows repository images in search results and organization pages. Use the logo consistently across:
- Repository description
- Topics/tags
- Organization profile

## Logo Specifications

### Dimensions
- **Aspect Ratio**: ~16:5 (wide panoramic)
- **Minimum Width**: 800px (for clarity)
- **Recommended Width**: 2000px (for high-DPI displays)
- **Social Preview**: 1280x640px (GitHub requirement)

### Color Scheme

**Light Mode Logo:**
- Background: Light/white
- Mushrooms: Warm tones (browns, oranges)
- Glow: Soft bioluminescent blue/green
- Network lines: Visible but subtle

**Dark Mode Logo:**
- Background: Dark/black
- Mushrooms: Same warm tones
- Glow: Brighter, more prominent bioluminescence
- Network lines: More visible, glowing

### File Format
- **PNG** with transparency (preferred)
- **SVG** for scalability (future enhancement)
- **WebP** for web optimization (future enhancement)

## Brand Guidelines

### Do ✅

- Use the adaptive logo in all documentation
- Maintain aspect ratio when resizing
- Ensure sufficient contrast with background
- Use the full logo with all three mushrooms

### Don't ❌

- Don't crop or separate individual mushrooms
- Don't change colors or add filters
- Don't place on busy backgrounds
- Don't stretch or distort the logo
- Don't use low-resolution versions

## Alternative Formats

### ASCII Art (CLI)

For terminal output where images aren't supported, use this ASCII representation:

```
    🍄  🍄  🍄
    ┊╲  ╱┊╲  ╱┊
    ┊ ╲╱ ┊ ╲╱ ┊
    ╰─────────╯
     mycelium
```

### Emoji Shorthand

In informal communication: 🍄 ✨ (mushroom + sparkles)

### Text Logo

For plain text contexts:
```
M Y C E L I U M
The underground network for AWS compute
```

## File Naming Convention

All logo files should follow this pattern:
- `logo-light.{ext}` - Light mode variant
- `logo-dark.{ext}` - Dark mode variant
- `logo-social.{ext}` - Social media preview (1280x640)
- `logo-favicon.{ext}` - Favicon (32x32, 64x64)
- `logo-icon.{ext}` - App icon (various sizes)

## Implementation Checklist

When adding a new logo:

1. [ ] Create both light and dark variants
2. [ ] Optimize file size (compress PNGs)
3. [ ] Test on both light and dark GitHub themes
4. [ ] Verify dimensions and aspect ratio
5. [ ] Add alt text for accessibility
6. [ ] Update this documentation
7. [ ] Commit logo files to repository
8. [ ] Update CHANGELOG if part of release

## License

The Mycelium logo is part of the Mycelium project and follows the same license as the codebase (MIT License).

## Questions?

For logo design questions or requests for additional formats, please open an issue on GitHub.
