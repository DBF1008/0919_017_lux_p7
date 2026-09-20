package subtitle

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

// Track is a subtitle file ready to be embedded into a video.
type Track struct {
	Path     string
	Language string
}

// EmbedStrategy embeds subtitle tracks into a video file.
// Implementations can be registered and plugged into a Processor
// through Options.Strategy.
type EmbedStrategy interface {
	Embed(videoPath string, tracks []Track) error
}

// FFmpegStrategy is the default EmbedStrategy, embedding subtitles
// with ffmpeg.
type FFmpegStrategy struct {
	// FFmpegPath optionally overrides the ffmpeg executable path.
	FFmpegPath string
}

func (s FFmpegStrategy) executable() string {
	if s.FFmpegPath != "" {
		return s.FFmpegPath
	}
	ffmpegFileName := "ffmpeg"
	if runtime.GOOS == "windows" {
		ffmpegFileName = "ffmpeg.exe"
	}
	// look for ffmpeg in the current directory first
	matches, err := filepath.Glob("./" + ffmpegFileName)
	if err == nil && len(matches) > 0 {
		return "./" + ffmpegFileName
	}
	return ffmpegFileName
}

// ISO 639-1 to ISO 639-2 language code mapping
var langToISO = map[string]string{
	"aa": "aar", "ab": "abk", "af": "afr", "ak": "aka", "am": "amh",
	"ar": "ara", "an": "arg", "as": "asm", "av": "ava", "ay": "aym",
	"az": "aze", "ba": "bak", "be": "bel", "bg": "bul", "bh": "bih",
	"bi": "bis", "bm": "bam", "bn": "ben", "bo": "tib", "br": "bre",
	"bs": "bos", "ca": "cat", "ce": "che", "ch": "cha", "co": "cos",
	"cr": "cre", "cs": "cze", "cu": "chu", "cv": "chv", "cy": "wel",
	"da": "dan", "de": "ger", "dv": "div", "dz": "dzo", "ee": "ewe",
	"el": "gre", "en": "eng", "eo": "epo", "es": "spa", "et": "est",
	"eu": "baq", "fa": "per", "ff": "ful", "fi": "fin", "fj": "fij",
	"fo": "fao", "fr": "fre", "fy": "fry", "ga": "gle", "gd": "gla",
	"gl": "glg", "gn": "grn", "gu": "guj", "gv": "glv", "ha": "hau",
	"he": "heb", "hi": "hin", "ho": "hmo", "hr": "hrv", "ht": "hat",
	"hu": "hun", "hy": "arm", "hz": "her", "ia": "ina", "id": "ind",
	"ie": "ile", "ig": "ibo", "ii": "iii", "ik": "ipk", "io": "ido",
	"is": "ice", "it": "ita", "iu": "iku", "ja": "jpn", "jv": "jav",
	"ka": "geo", "kg": "kon", "ki": "kik", "kj": "kua", "kk": "kaz",
	"kl": "kal", "km": "khm", "kn": "kan", "ko": "kor", "kr": "kau",
	"ks": "kas", "ku": "kur", "kv": "kom", "kw": "cor", "ky": "kir",
	"la": "lat", "lb": "ltz", "lg": "lug", "li": "lim", "ln": "lin",
	"lo": "lao", "lt": "lit", "lu": "lub", "lv": "lav", "mg": "mlg",
	"mh": "mah", "mi": "mao", "mk": "mac", "ml": "mal", "mn": "mon",
	"mr": "mar", "ms": "may", "mt": "mlt", "my": "bur", "na": "nau",
	"nb": "nob", "nd": "nde", "ne": "nep", "ng": "ndo", "nl": "nld",
	"nn": "nno", "no": "nor", "nr": "nbl", "nv": "nav", "ny": "nya",
	"oc": "oci", "oj": "oji", "om": "orm", "or": "ori", "os": "oss",
	"pa": "pan", "pi": "pli", "pl": "pol", "ps": "pus", "pt": "por",
	"qu": "que", "rm": "roh", "rn": "run", "ro": "rum", "ru": "rus",
	"rw": "kin", "sa": "san", "sc": "srd", "sd": "snd", "se": "sme",
	"sg": "sag", "si": "sin", "sk": "slo", "sl": "slv", "sm": "smo",
	"sn": "sna", "so": "som", "sq": "alb", "sr": "srp", "ss": "ssw",
	"st": "sot", "su": "sun", "sv": "swe", "sw": "swa", "ta": "tam",
	"te": "tel", "tg": "tgk", "th": "tha", "ti": "tir", "tk": "tuk",
	"tl": "tgl", "tn": "tsn", "to": "ton", "tr": "tur", "ts": "tso",
	"tt": "tat", "tw": "twi", "ty": "tah", "ug": "uig", "uk": "ukr",
	"ur": "urd", "uz": "uzb", "ve": "ven", "vi": "vie", "vo": "vol",
	"wa": "wln", "wo": "wol", "xh": "xho", "yi": "yid", "yo": "yor",
	"za": "zha", "zh": "chi", "zu": "zul",
}

// ToISO639 converts a language code (eg: "en-US") to ISO 639-2 (eg: "eng").
func ToISO639(lang string) string {
	base := strings.ToLower(lang)
	if i := strings.IndexAny(base, "-_"); i != -1 {
		base = base[:i]
	}
	if iso, ok := langToISO[base]; ok {
		return iso
	}
	if len(base) == 3 {
		return base
	}
	return "und"
}

// subtitleCodec returns the appropriate subtitle codec for the container format
func subtitleCodec(ext string) string {
	switch strings.ToLower(ext) {
	case ".mp4":
		return "mov_text"
	case ".webm":
		return "webvtt"
	default:
		return ""
	}
}

// args builds the ffmpeg arguments for embedding the tracks into the video.
func (s FFmpegStrategy) args(videoPath, tempOutput string, tracks []Track) []string {
	cmds := []string{"-y", "-i", videoPath}
	for _, track := range tracks {
		cmds = append(cmds, "-i", track.Path)
	}

	cmds = append(cmds,
		"-map", "0", "-dn", "-ignore_unknown",
		"-c", "copy",
	)

	if codec := subtitleCodec(filepath.Ext(videoPath)); codec != "" {
		cmds = append(cmds, "-c:s", codec)
	}

	// Exclude existing subtitles, then map new ones
	cmds = append(cmds, "-map", "-0:s")
	for i, track := range tracks {
		iso := ToISO639(track.Language)
		cmds = append(cmds,
			"-map", fmt.Sprintf("%d:0", i+1),
			fmt.Sprintf("-metadata:s:s:%d", i), fmt.Sprintf("language=%s", iso),
			fmt.Sprintf("-metadata:s:s:%d", i), fmt.Sprintf("handler_name=%s", track.Language),
			fmt.Sprintf("-metadata:s:s:%d", i), fmt.Sprintf("title=%s", track.Language),
		)
	}

	return append(cmds, tempOutput)
}

// Embed embeds the subtitle tracks into the video with ffmpeg.
func (s FFmpegStrategy) Embed(videoPath string, tracks []Track) error {
	ext := filepath.Ext(videoPath)
	tempOutput := videoPath + ".temp" + ext

	cmd := exec.Command(s.executable(), s.args(videoPath, tempOutput, tracks)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		os.Remove(tempOutput) // nolint
		return errors.Errorf("%s\n%s", err, stderr.String())
	}
	// the rename below can not overwrite an existing file on Windows
	os.Remove(videoPath) // nolint
	return os.Rename(tempOutput, videoPath)
}
