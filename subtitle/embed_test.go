package subtitle

import (
	"reflect"
	"testing"
)

func TestToISO639(t *testing.T) {
	tests := []struct {
		lang string
		want string
	}{
		{"en", "eng"},
		{"en-US", "eng"},
		{"zh_CN", "chi"},
		{"zh-Hans", "chi"},
		{"fr", "fre"},
		{"de", "ger"},
		{"ar", "ara"},
		{"hi", "hin"},
		{"th", "tha"},
		{"vi", "vie"},
		{"uk", "ukr"},
		{"tr", "tur"},
		{"eng", "eng"}, // already ISO 639-2
		{"und", "und"}, // already ISO 639-2
		{"xx", "und"},  // unknown
		{"", "und"},    // empty
	}
	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			if got := ToISO639(tt.lang); got != tt.want {
				t.Errorf("ToISO639(%q) = %q, want %q", tt.lang, got, tt.want)
			}
		})
	}
}

func TestFFmpegStrategyArgs(t *testing.T) {
	strategy := FFmpegStrategy{}
	tracks := []Track{
		{Path: "en.srt", Language: "en"},
		{Path: "zh.srt", Language: "zh-CN"},
	}
	got := strategy.args("video.mp4", "video.mp4.temp.mp4", tracks)
	want := []string{
		"-y", "-i", "video.mp4", "-i", "en.srt", "-i", "zh.srt",
		"-map", "0", "-dn", "-ignore_unknown",
		"-c", "copy",
		"-c:s", "mov_text",
		"-map", "-0:s",
		"-map", "1:0",
		"-metadata:s:s:0", "language=eng",
		"-metadata:s:s:0", "handler_name=en",
		"-metadata:s:s:0", "title=en",
		"-map", "2:0",
		"-metadata:s:s:1", "language=chi",
		"-metadata:s:s:1", "handler_name=zh-CN",
		"-metadata:s:s:1", "title=zh-CN",
		"video.mp4.temp.mp4",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("args() =\n%v\nwant:\n%v", got, want)
	}
}

func TestFFmpegStrategyArgsNoCodec(t *testing.T) {
	strategy := FFmpegStrategy{}
	tracks := []Track{{Path: "en.srt", Language: "en"}}
	got := strategy.args("video.mkv", "video.mkv.temp.mkv", tracks)
	for _, arg := range got {
		if arg == "-c:s" {
			t.Errorf("args() should not contain -c:s for mkv, got %v", got)
		}
	}
}
