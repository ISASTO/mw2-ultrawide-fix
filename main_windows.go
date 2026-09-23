//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const appVersion = "1.0.0"

var (
	resolutionCombo syscall.Handle
	statusLabel     syscall.Handle
	mainWindow      syscall.Handle
	appDir          string
)

func message(title, body string, flags uintptr) {
	procMessageBoxW.Call(uintptr(mainWindow), uintptr(unsafe.Pointer(ptr(body))), uintptr(unsafe.Pointer(ptr(title))), flags)
}

func parseResolution(s string) (int, int, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "×", "x")
	s = strings.ReplaceAll(s, " ", "")
	parts := strings.Split(s, "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("enter a resolution like 3440x1440")
	}
	w, err := strconv.Atoi(parts[0])
	if err != nil || w <= 0 {
		return 0, 0, fmt.Errorf("invalid width")
	}
	h, err := strconv.Atoi(parts[1])
	if err != nil || h <= 0 {
		return 0, 0, fmt.Errorf("invalid height")
	}
	return w, h, nil
}

func wndProc(hwnd syscall.Handle, msg uint32, wparam, lparam uintptr) uintptr {
	switch msg {
	case WM_COMMAND:
		switch int(wparam & 0xffff) {
		case IDC_APPLY:
			w, h, err := parseResolution(getWindowText(resolutionCombo))
			if err != nil {
				message("Invalid resolution", err.Error(), MB_OK|MB_ICONERROR)
				return 0
			}
			setText(statusLabel, "Applying patch...")
			result, err := applyPatch(appDir, w, h)
			if err != nil {
				setText(statusLabel, "Patch not applied.")
				message("Could not apply fix", err.Error(), MB_OK|MB_ICONERROR)
				return 0
			}
			setText(statusLabel, fmt.Sprintf("Fixed for %dx%d. Backup saved.", w, h))
			message("MW2 Ultrawide Fix", fmt.Sprintf("Done!\n\nPatched %d aspect-ratio value(s) for %dx%d (%.4f:1).\n\nYour original iw4sp.exe is backed up automatically.", result.Count, w, h, result.AspectRatio), MB_OK|MB_ICONINFORMATION)
			return 0

		case IDC_RESTORE:
			setText(statusLabel, "Restoring original executable...")
			if err := restoreOriginal(appDir); err != nil {
				setText(statusLabel, "Restore not completed.")
				message("Could not restore", err.Error(), MB_OK|MB_ICONERROR)
				return 0
			}
			setText(statusLabel, "Original iw4sp.exe restored.")
			message("MW2 Ultrawide Fix", "Original iw4sp.exe restored successfully.", MB_OK|MB_ICONINFORMATION)
			return 0
		}
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wparam, lparam)
	return ret
}

func detectResolution() (int, int) {
	w, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	h, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	return int(w), int(h)
}

func main() {
	procSetProcessDPIAware.Call()

	self, err := os.Executable()
	if err != nil {
		return
	}
	appDir = filepath.Dir(self)

	hInstance, _, _ := procGetModuleHandleW.Call(0)
	className := ptr("MW2UltrawideFixWindow")
	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   syscall.NewCallback(wndProc),
		HInstance:     syscall.Handle(hInstance),
		HbrBackground: syscall.Handle(COLOR_WINDOW + 1),
		LpszClassName: className,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(ptr("MW2 Ultrawide Fix"))),
		WS_OVERLAPPED|WS_CAPTION|WS_SYSMENU|WS_MINIMIZEBOX,
		CW_USEDEFAULT, CW_USEDEFAULT, 500, 275,
		0, 0, hInstance, 0,
	)
	if hwnd == 0 {
		return
	}
	mainWindow = syscall.Handle(hwnd)

	createControl("STATIC", "Call of Duty: Modern Warfare 2 (2009)", WS_CHILD|WS_VISIBLE|SS_LEFT, 24, 20, 430, 24, mainWindow, 0)
	createControl("STATIC", "Select your screen resolution. Your primary display is detected automatically.", WS_CHILD|WS_VISIBLE|SS_LEFT, 24, 50, 440, 36, mainWindow, 0)

	detectedW, detectedH := detectResolution()
	createControl("STATIC", "Resolution:", WS_CHILD|WS_VISIBLE|SS_LEFT, 24, 94, 80, 22, mainWindow, 0)
	resolutionCombo = createControl("COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_VSCROLL|CBS_DROPDOWN, 110, 90, 190, 220, mainWindow, IDC_RESOLUTION)

	resolutions := []string{
		fmt.Sprintf("%dx%d", detectedW, detectedH),
		"2560x1080", "3440x1440", "3840x1600", "3840x1080",
		"5120x1440", "5120x2160", "7680x2160",
	}
	seen := map[string]bool{}
	index := 0
	for _, r := range resolutions {
		if seen[r] {
			continue
		}
		seen[r] = true
		procSendMessageW.Call(uintptr(resolutionCombo), CB_ADDSTRING, 0, uintptr(unsafe.Pointer(ptr(r))))
		if r == fmt.Sprintf("%dx%d", detectedW, detectedH) {
			index = len(seen) - 1
		}
	}
	procSendMessageW.Call(uintptr(resolutionCombo), CB_SETCURSEL, uintptr(index), 0)

	createControl("BUTTON", "Apply Fix", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 24, 135, 210, 38, mainWindow, IDC_APPLY)
	createControl("BUTTON", "Restore Original", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 250, 135, 210, 38, mainWindow, IDC_RESTORE)

	status := "Ready. Place this program next to iw4sp.exe."
	if _, err := os.Stat(filepath.Join(appDir, targetExeName)); err == nil {
		status = "Ready. iw4sp.exe found."
	}
	statusLabel = createControl("STATIC", status, WS_CHILD|WS_VISIBLE|SS_LEFT, 24, 190, 430, 22, mainWindow, 0)
	createControl("STATIC", "v"+appVersion+"  •  Backup is created automatically", WS_CHILD|WS_VISIBLE|SS_LEFT, 24, 217, 430, 20, mainWindow, 0)

	screenW, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	screenH, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	x := (int(screenW) - 500) / 2
	y := (int(screenH) - 275) / 2
	procSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), 500, 275, 0x0004|0x0010)
	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)

	var msg MSG
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}
