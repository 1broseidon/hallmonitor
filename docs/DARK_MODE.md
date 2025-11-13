# Dark Mode

The Hall Monitor dashboard comes with **dark mode enabled by default**, providing a comfortable viewing experience in low-light environments and reducing eye strain.

## Features

- ✨ **Dark Mode by Default** - Dashboard loads in dark theme automatically
- 🌓 **Theme Toggle** - Switch between dark and light modes with one click
- 💾 **Persistent** - Theme preference is saved to browser localStorage
- 🎨 **Beautiful Colors** - Carefully chosen colors for both themes
- ⚡ **Instant Switch** - Theme changes apply immediately without page reload

## Using Dark Mode

### Default Behavior
The dashboard automatically loads in dark mode. You'll see:
- Dark backgrounds (#1a1a1a primary, #2d2d2d for cards)
- Light text (#e0e0e0 primary)
- Subtle shadows and borders for depth
- Vibrant accent colors (green, red, blue, yellow) that pop against dark background

### Switching Themes

Click the **theme toggle button** (⊙ circle half-stroke icon) in the top-right next to the Refresh button:

```
[⊙] [🔄 Refresh] [💾 Export to Grafana]
```

- **First click**: Switches to light mode
- **Second click**: Switches back to dark mode
- Your choice is **saved** and persists across browser sessions

## Color Palettes

### Dark Mode (Default)
```
Background:     #1a1a1a (very dark gray)
Cards:          #2d2d2d (dark gray)
Text:           #e0e0e0 (light gray)
Accent:         #667eea to #764ba2 (purple gradient)
Success:        #48c78e (green)
Error:          #f14668 (red)
Warning:        #ffdd57 (yellow)
Info:           #3298dc (blue)
```

### Light Mode
```
Background:     #f5f5f5 (light gray)
Cards:          #ffffff (white)
Text:           #222222 (dark gray)
Accent:         #667eea to #764ba2 (purple gradient)
Success:        #48c78e (green)
Error:          #f14668 (red)
Warning:        #ffdd57 (yellow)
Info:           #3298dc (blue)
```

## Browser Compatibility

Theme persistence works in all modern browsers:
- Chrome/Edge 4+
- Firefox 3.5+
- Safari 4+
- Opera 10+

Older browsers will show dark mode but won't remember preference.

## Preferences

Your theme choice is stored in browser localStorage:
- **Key**: `theme`
- **Values**: `"dark"` or `"light"`
- **Persistence**: Until browser localStorage is cleared

To reset theme preference:
```javascript
// In browser console:
localStorage.removeItem('theme');
location.reload();
```

## Accessibility

Both themes meet WCAG AA contrast requirements:

**Dark Mode:**
- Text contrast: 13.5:1 (#e0e0e0 on #1a1a1a)
- Status colors easily distinguishable

**Light Mode:**
- Text contrast: 9.8:1 (#222 on #f5f5f5)
- Status colors easily distinguishable

## Pro Tips

1. **System Preference**: The theme doesn't auto-follow OS dark mode (yet) - click the button to switch
2. **Mobile Friendly**: Theme toggle works great on mobile/tablet too
3. **Screenshot Friendly**: Take screenshots in whichever theme works best for your docs
4. **Performance**: Theme switching has zero performance impact

## Troubleshooting

**Theme not saving?**
- Check if localStorage is enabled in browser settings
- Try a different browser
- Clear browser cache and try again

**Colors look wrong?**
- Check if you're in private/incognito mode (localStorage may be disabled)
- Update your browser to the latest version
- Try the other theme and switch back

**Prefer always light mode?**
- Use the theme toggle and your preference will be saved
- Or clear localStorage and stay in light mode

---

**Questions?** The theme system is simple, CSS-based, and works offline!
