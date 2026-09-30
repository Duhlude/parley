package main

// Parley's logo, embedded as icon resources (rsrc_windows_amd64.syso,
// built from parley.rc): 1 = the badge, 2 = the speech bubbles alone.

// resIcon loads icon resource id at size px (cached). 0 if unavailable.
func resIcon(id, px int) uintptr {
	k := [2]int{id, px}
	if h, ok := iconCache[k]; ok {
		return h
	}
	inst, _, _ := pGetModuleHandleW.Call(0)
	h, _, _ := pLoadImageW.Call(inst, uintptr(id), 1 /*IMAGE_ICON*/, uintptr(px), uintptr(px), 0)
	iconCache[k] = h
	return h
}

// drawIcon paints an icon resource into r; false if it couldn't.
func drawIcon(hdc uintptr, id int, r RECT) bool {
	w := r.Right - r.Left
	h := resIcon(id, int(w))
	if h == 0 {
		return false
	}
	pDrawIconEx.Call(hdc, uintptr(r.Left), uintptr(r.Top), h, uintptr(w), uintptr(r.Bottom-r.Top), 0, 0, 3 /*DI_NORMAL*/)
	return true
}

// appIcons returns the window/tray icons: the logo, falling back to the
// icon drawn in code if the resource is missing (e.g. a dev build).
func appIcons() (big, small uintptr) {
	bs, _, _ := pGetSystemMetrics.Call(11) // SM_CXICON
	ss, _, _ := pGetSystemMetrics.Call(49) // SM_CXSMICON
	big, small = resIcon(1, int(bs)), resIcon(1, int(ss))
	if big == 0 || small == 0 {
		ic := makeIcon()
		return ic, ic
	}
	return big, small
}
