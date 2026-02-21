package imagecdn

import "testing"

func TestFlatCDNURL_IgnoresTransforms(t *testing.T) {
	cdn := New("https://cdn.example.com/assets")
	got, err := cdn.URL(Request{
		AssetID:   "001",
		Extension: ".png",
		Crop: &Crop{
			X:        0,
			Y:        0,
			WidthPX:  512,
			HeightPX: 768,
		},
		Delivery: &Delivery{WidthPX: 192},
	})
	if err != nil {
		t.Fatalf("resolve url: %v", err)
	}
	want := "https://cdn.example.com/assets/001.png"
	if got != want {
		t.Fatalf("cdn.URL(...) = %q, want %q", got, want)
	}
}

func TestCloudinaryCDNURL_IncludesCropAndDeliveryTransforms(t *testing.T) {
	cdn := New("https://res.cloudinary.com/fracturing-space/image/upload")
	got, err := cdn.URL(Request{
		AssetID:   "001",
		Extension: ".png",
		Crop: &Crop{
			X:        0,
			Y:        0,
			WidthPX:  512,
			HeightPX: 768,
		},
		Delivery: &Delivery{WidthPX: 192},
	})
	if err != nil {
		t.Fatalf("resolve url: %v", err)
	}
	want := "https://res.cloudinary.com/fracturing-space/image/upload/c_crop,w_512,h_768,x_0,y_0/f_auto,q_auto,dpr_auto,c_limit,w_192/001.png"
	if got != want {
		t.Fatalf("cdn.URL(...) = %q, want %q", got, want)
	}
}

func TestCDNURL_RejectsMissingAssetID(t *testing.T) {
	cdn := New("https://cdn.example.com/assets")
	_, err := cdn.URL(Request{})
	if err != ErrAssetIDRequired {
		t.Fatalf("cdn.URL(...) error = %v, want %v", err, ErrAssetIDRequired)
	}
}
