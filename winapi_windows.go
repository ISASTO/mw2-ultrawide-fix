//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

const (
	WS_OVERLAPPED       = 0x00000000
	WS_CAPTION          = 0x00C00000
	WS_SYSMENU          = 0x00080000
	WS_MINIMIZEBOX      = 0x00020000
	WS_VISIBLE          = 0x10000000
	WS_CHILD            = 0x40000000
	WS_TABSTOP          = 0x00010000
	WS_VSCROLL          = 0x00200000
	CBS_DROPDOWN        = 0x0002
	BS_PUSHBUTTON       = 0x00000000
	SS_LEFT             = 0x00000000
	WM_DESTROY          = 0x0002
	WM_COMMAND          = 0x0111
	WM_SETFONT          = 0x0030
	CB_ADDSTRING        = 0x0143
	CB_SETCURSEL        = 0x014E
	SW_SHOW             = 5
	CW_USEDEFAULT       = 0x80000000
	MB_OK               = 0x00000000
	MB_ICONINFORMATION  = 0x00000040
	MB_ICONERROR        = 0x00000010
	SM_CXSCREEN         = 0
	SM_CYSCREEN         = 1
	DEFAULT_GUI_FONT    = 17
	COLOR_WINDOW        = 5
	IDC_APPLY           = 1001
	IDC_RESTORE         = 1002
	IDC_RESOLUTION      = 1003
)

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     syscall.Handle
	HIcon         syscall.Handle
	HCursor       syscall.Handle
	HbrBackground syscall.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       syscall.Handle
}

type MSG struct {
	HWnd    syscall.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")

	procRegisterClassExW   = user32.NewProc("RegisterClassExW")
	procCreateWindowExW    = user32.NewProc("CreateWindowExW")
	procDefWindowProcW     = user32.NewProc("DefWindowProcW")
	procShowWindow         = user32.NewProc("ShowWindow")
	procUpdateWindow       = user32.NewProc("UpdateWindow")
	procGetMessageW        = user32.NewProc("GetMessageW")
	procTranslateMessage   = user32.NewProc("TranslateMessage")
	procDispatchMessageW   = user32.NewProc("DispatchMessageW")
	procPostQuitMessage    = user32.NewProc("PostQuitMessage")
	procSendMessageW       = user32.NewProc("SendMessageW")
	procMessageBoxW        = user32.NewProc("MessageBoxW")
	procGetSystemMetrics   = user32.NewProc("GetSystemMetrics")
	procSetWindowTextW     = user32.NewProc("SetWindowTextW")
	procGetWindowTextW     = user32.NewProc("GetWindowTextW")
	procSetWindowPos       = user32.NewProc("SetWindowPos")
	procSetProcessDPIAware = user32.NewProc("SetProcessDPIAware")
	procGetModuleHandleW   = kernel32.NewProc("GetModuleHandleW")
	procGetStockObject     = gdi32.NewProc("GetStockObject")
)

func ptr(s string) *uint16 { return syscall.StringToUTF16Ptr(s) }

func createControl(class, text string, style uint32, x, y, w, h int32, parent syscall.Handle, id int) syscall.Handle {
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(ptr(class))),
		uintptr(unsafe.Pointer(ptr(text))),
		uintptr(style),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		uintptr(parent), uintptr(id), 0, 0,
	)
	if hwnd != 0 {
		font, _, _ := procGetStockObject.Call(DEFAULT_GUI_FONT)
		procSendMessageW.Call(hwnd, WM_SETFONT, font, 1)
	}
	return syscall.Handle(hwnd)
}

func setText(hwnd syscall.Handle, text string) {
	procSetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(ptr(text))))
}

func getWindowText(hwnd syscall.Handle) string {
	buf := make([]uint16, 128)
	procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}
