package cli

import (
	"testing"
)

func TestOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		wantErr bool
	}{
		{
			name: "valid options",
			opts: Options{
				InputPaths: []string{"image.jpg"},
				Quality:    80,
				Suffix:     "_compressed",
			},
			wantErr: false,
		},
		{
			name: "no input files",
			opts: Options{
				InputPaths: []string{},
				Quality:    80,
				Suffix:     "_compressed",
			},
			wantErr: true,
		},
		{
			name: "quality too low",
			opts: Options{
				InputPaths: []string{"image.jpg"},
				Quality:    0,
				Suffix:     "_compressed",
			},
			wantErr: true,
		},
		{
			name: "quality too high",
			opts: Options{
				InputPaths: []string{"image.jpg"},
				Quality:    101,
				Suffix:     "_compressed",
			},
			wantErr: true,
		},
		{
			name: "empty suffix",
			opts: Options{
				InputPaths: []string{"image.jpg"},
				Quality:    80,
				Suffix:     "",
			},
			wantErr: true,
		},
		{
			name: "version flag skips validation",
			opts: Options{
				InputPaths: []string{},
				Quality:    0,
				Suffix:     "",
				Version:    true,
			},
			wantErr: false,
		},
		{
			name: "multiple input files",
			opts: Options{
				InputPaths: []string{"image1.jpg", "image2.png"},
				Quality:    80,
				Suffix:     "_compressed",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultValues(t *testing.T) {
	if DefaultQuality != 80 {
		t.Errorf("DefaultQuality = %d, want 80", DefaultQuality)
	}
	if DefaultSuffix != "_compressed" {
		t.Errorf("DefaultSuffix = %q, want %q", DefaultSuffix, "_compressed")
	}
}
