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

// DefaultEmbedStrategy is the name of the built-in ffmpeg embed strategy.
const DefaultEmbedStrategy = "ffmpeg"

// EmbedStrategy embeds subtitle tracks into a video file.
type EmbedStrategy interface {
	// Name returns the strategy name used for registration.
	Name() string
	// Embed muxes the given subtitle tracks into the video at videoPath.
	Embed(videoPath string, tracks []Track) error
}

// RegisterEmbedStrategy registers an embed strategy under its name,
// replacing any existing strategy with the same name.
func (p *Processor) RegisterEmbedStrategy(name string, s EmbedStrategy) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.strategies[name] = s
}

// SetEmbedStrategy selects the embed strategy used by Embed.
func (p *Processor) SetEmbedStrategy(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.strategies[name]; !ok {
		return fmt.Errorf("no subtitle embed strategy registered under name %q", name)
	}
	p.embedderName = name
	return nil
}

// Embed muxes the subtitle tracks into the video using the active
// embed strategy.
func (p *Processor) Embed(videoPath string, tracks []Track) error {
	p.mu.Lock()
	strategy, ok := p.strategies[p.embedderName]
	p.mu.Unlock()
	if !ok {
		return fmt.Errorf("no subtitle embed strategy registered under name %q", p.embedderName)
	}
	return strategy.Embed(videoPath, tracks)
}

// ffmpegEmbedStrategy embeds subtitles with ffmpeg.
type ffmpegEmbedStrategy struct{}

// NewFFmpegEmbedStrategy returns the ffmpeg based EmbedStrategy.
func NewFFmpegEmbedStrategy() EmbedStrategy {
	return ffmpegEmbedStrategy{}
}

func (ffmpegEmbedStrategy) Name() string { return DefaultEmbedStrategy }

func (ffmpegEmbedStrategy) Embed(videoPath string, tracks []Track) error {
	ext := filepath.Ext(videoPath)
	tempOutput := videoPath + ".temp" + ext

	cmds := []string{"-y", "-i", videoPath}
	for _, track := range tracks {
		cmds = append(cmds, "-i", track.Path)
	}

	cmds = append(cmds,
		"-map", "0", "-dn", "-ignore_unknown",
		"-c", "copy",
	)

	if codec := subtitleCodec(ext); codec != "" {
		cmds = append(cmds, "-c:s", codec)
	}

	// Exclude existing subtitles, then map the new ones.
	cmds = append(cmds, "-map", "-0:s")
	for i, track := range tracks {
		cmds = append(cmds,
			"-map", fmt.Sprintf("%d:0", i+1),
			fmt.Sprintf("-metadata:s:s:%d", i), fmt.Sprintf("language=%s", ToISO639(track.Lang)),
			fmt.Sprintf("-metadata:s:s:%d", i), fmt.Sprintf("handler_name=%s", track.Lang),
			fmt.Sprintf("-metadata:s:s:%d", i), fmt.Sprintf("title=%s", track.Lang),
		)
	}

	cmds = append(cmds, tempOutput)

	var stderr bytes.Buffer
	cmd := exec.Command(FindFFmpegExecutable(), cmds...)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		os.Remove(tempOutput) // nolint
		return errors.Errorf("%s\n%s", err, stderr.String())
	}
	// Remove the original file first: renaming over an existing file
	// fails on Windows.
	os.Remove(videoPath) // nolint
	return os.Rename(tempOutput, videoPath)
}

// FindFFmpegExecutable locates the ffmpeg executable, preferring one in
// the current directory over the one in PATH.
func FindFFmpegExecutable() string {
	ffmpegFileName := "ffmpeg"
	if runtime.GOOS == "windows" {
		ffmpegFileName = "ffmpeg.exe"
	}
	matches, err := filepath.Glob("./" + ffmpegFileName)
	if err == nil && len(matches) > 0 {
		return "./" + ffmpegFileName
	}
	return ffmpegFileName
}

// subtitleCodec returns the appropriate subtitle codec for the container format.
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

// langToISO maps ISO 639-1 language codes to ISO 639-2/T (terminology) codes.
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
	"nb": "nob", "nd": "nde", "ne": "nep", "ng": "ndo", "nl": "dut",
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

// ToISO639 converts a language code (e.g. "en" or "en-US") to its
// ISO 639-2 representation (e.g. "eng"). Unknown codes of length 3 are
// assumed to already be ISO 639-2; anything else maps to "und".
func ToISO639(lang string) string {
	base := strings.ToLower(strings.TrimSpace(lang))
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
