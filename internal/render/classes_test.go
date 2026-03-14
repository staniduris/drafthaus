package render

import "testing"

func TestMergeClasses(t *testing.T) {
	tests := []struct{ name, base, extra, want string }{
		{"base only", "dh-section", "", "dh-section"},
		{"extra only", "", "bg-white p-4", "bg-white p-4"},
		{"both", "dh-card", "bg-white shadow-lg", "dh-card bg-white shadow-lg"},
		{"trims", " dh-card ", " bg-white ", "dh-card bg-white"},
		{"empty", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergeClasses(tt.base, tt.extra); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSanitizeClasses(t *testing.T) {
	tests := []struct{ input, want string }{
		{"bg-white p-4", "bg-white p-4"},
		{"text-[#fff] w-[300px]", "text-[#fff] w-[300px]"},
		{"hover:bg-red-500", "hover:bg-red-500"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := sanitizeClasses(tt.input); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
