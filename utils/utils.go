package utils

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"github.com/yamamushi/kmud-2020/color"
	"io"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/yamamushi/kmud-2020/types"
)

type Prompter interface {
	GetPrompt() string
}

type simplePrompter struct {
	prompt string
}

func (sp simplePrompter) GetPrompt() string {
	return sp.prompt
}

// SimpleRompter returns a Prompter that always returns the given string as its prompt
func SimplePrompter(prompt string) Prompter {
	var prompter simplePrompter
	prompter.prompt = prompt
	return &prompter
}

func Write(conn io.Writer, text string, cm color.ColorMode) error {
	_, err := conn.Write([]byte(color.ProcessColors(text, cm)))
	return err
}

func WriteLine(conn io.Writer, line string, cm color.ColorMode) error {
	return Write(conn, line+"\r\n", cm)
}

// ClearLine sends the VT100 code for erasing the line followed by a carriage
// return to move the cursor back to the beginning of the line
func ClearLine(conn io.Writer) error {
	clearline := "\x1B[2K"
	return Write(conn, clearline+"\r", color.ModeNone)
}

func Simplify(str string) string {
	simpleStr := strings.TrimSpace(str)
	simpleStr = strings.ToLower(simpleStr)
	return simpleStr
}

func GetRawUserInputSuffix(conn io.ReadWriter, prompt string, suffix string, cm color.ColorMode) string {
	return GetRawUserInputSuffixP(conn, SimplePrompter(prompt), suffix, cm)
}

func GetRawUserInputSuffixP(conn io.ReadWriter, prompter Prompter, suffix string, cm color.ColorMode) string {
	scanner := bufio.NewScanner(conn)

	for {
		Write(conn, prompter.GetPrompt(), cm)

		if !scanner.Scan() {
			err := scanner.Err()
			if err == nil {
				err = io.EOF
			}

			panic(err)
		}

		input := scanner.Text()
		Write(conn, suffix, cm)

		if input == "x" || input == "X" {
			return ""
		} else if input != "" {
			return input
		}
	}
}

func GetRawUserInputP(conn io.ReadWriter, prompter Prompter, cm color.ColorMode) string {
	return GetRawUserInputSuffixP(conn, prompter, "", cm)
}

func GetRawUserInput(conn io.ReadWriter, prompt string, cm color.ColorMode) string {
	return GetRawUserInputP(conn, SimplePrompter(prompt), cm)
}

func GetUserInputP(conn io.ReadWriter, prompter Prompter, cm color.ColorMode) string {
	input := GetRawUserInputP(conn, prompter, cm)
	return Simplify(input)
}

func GetUserInput(conn io.ReadWriter, prompt string, cm color.ColorMode) string {
	input := GetUserInputP(conn, SimplePrompter(prompt), cm)
	return Simplify(input)
}

func Log(msg string) {
	log.Println("Log: ")
}
func Error(err string) {
	HandleError(errors.New(err))
}

func HandleError(err error) {
	if err != nil {
		log.Printf("Error: %s", err)
		//panic(err)
	}
}

func EmptyError() error {
	return errors.New("")
}

func FormatName(name string) string {
	if name == "" {
		return name
	}

	fields := strings.Fields(name)
	for i, field := range fields {
		runes := []rune(strings.ToLower(field))
		runes[0] = unicode.ToUpper(runes[0])
		fields[i] = string(runes)
	}

	return strings.Join(fields, " ")
}

func Argify(data string) (string, string) {
	fields := strings.Fields(data)

	if len(fields) == 0 {
		return "", ""
	}

	command := Simplify(fields[0])
	params := strings.TrimSpace(data[len(command):])

	return command, params
}

func rowEmpty(row string) bool {
	for _, char := range row {
		if char != ' ' {
			return false
		}
	}
	return true
}

func TrimUpperRows(rows []string) []string {
	for _, row := range rows {
		if !rowEmpty(row) {
			break
		}

		rows = rows[1:]
	}

	return rows
}

func TrimLowerRows(rows []string) []string {
	for i := len(rows) - 1; i >= 0; i -= 1 {
		row := rows[i]
		if !rowEmpty(row) {
			break
		}
		rows = rows[:len(rows)-1]
	}

	return rows
}

func TrimEmptyRows(str string) string {
	rows := strings.Split(str, "\r\n")
	return strings.Join(TrimLowerRows(TrimUpperRows(rows)), "\r\n")
}

func ValidateName(name string) error {
	const MinSize = 3
	const MaxSize = 12

	if len(name) < MinSize || len(name) > MaxSize {
		return errors.New(fmt.Sprintf("Names must be between %v and %v letters long", MinSize, MaxSize))
	}

	regex := regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9]*$")

	if !regex.MatchString(name) {
		return errors.New("Names may only contain letters or numbers (A-Z, 0-9), and must begin with a letter")
	}

	return nil
}

func MonitorChannel() {
	// TODO: See if there's a way to take in a generic channel and see how close it is to being full
}

// BestMatch searches the given list for the given pattern, the index of the
// longest match that starts with the given pattern is returned. Returns -1 if
// no match was found, -2 if the result is ambiguous. The search is case
// insensitive
func BestMatch(pattern string, searchList []string) int {
	pattern = strings.ToLower(pattern)

	index := -1

	for i, searchItem := range searchList {
		searchItem = strings.ToLower(searchItem)

		if searchItem == pattern {
			return i
		}

		if strings.HasPrefix(searchItem, pattern) {
			if index != -1 {
				return -2
			}

			index = i
		}
	}

	return index
}

func compress(data []byte) []byte {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(data)
	w.Close()
	return b.Bytes()
}

type WatchableReadWriter struct {
	rw       io.ReadWriter
	watchers []io.ReadWriter
}

func NewWatchableReadWriter(rw io.ReadWriter) *WatchableReadWriter {
	var watchable WatchableReadWriter
	watchable.rw = rw
	return &watchable
}

func (w *WatchableReadWriter) Read(p []byte) (int, error) {
	n, err := w.rw.Read(p)

	for _, watcher := range w.watchers {
		watcher.Write(p[:n])
	}

	return n, err
}

func (w *WatchableReadWriter) Write(p []byte) (int, error) {
	for _, watcher := range w.watchers {
		watcher.Write(p)
	}

	return w.rw.Write(p)
}

func (w *WatchableReadWriter) AddWatcher(rw io.ReadWriter) {
	w.watchers = append(w.watchers, rw)
}

func (w *WatchableReadWriter) RemoveWatcher(rw io.ReadWriter) {
	for i, watcher := range w.watchers {
		if watcher == rw {
			// TODO: Potential memory leak. See http://code.google.com/p/go-wiki/wiki/SliceTricks
			w.watchers = append(w.watchers[:i], w.watchers[i+1:]...)
			return
		}
	}
}

// Case-insensitive string comparison
func Compare(str1, str2 string) bool {
	return strings.ToLower(str1) == strings.ToLower(str2)
}

// Throttler is a simple utility class that allows events to occur on a
// deterministic recurring basis. Every call to Sync() will block until the
// duration of the Throttler's interval has passed since the last call to
// Sync()
type Throttler struct {
	lastTime time.Time
	interval time.Duration
}

func NewThrottler(interval time.Duration) *Throttler {
	var throttler Throttler
	throttler.lastTime = time.Now()
	throttler.interval = interval
	return &throttler
}

func (t *Throttler) Sync() {
	diff := time.Since(t.lastTime)
	if diff < t.interval {
		time.Sleep(t.interval - diff)
	}
	t.lastTime = time.Now()
}

// Random returns a random integer between low and high, inclusive
func Random(low, high int) int {
	if high < low {
		high, low = low, high
	}

	diff := high - low

	if diff == 0 {
		return low
	}

	result := rand.Int() % (diff + 1)
	result += low

	return result
}

func DirectionToExitString(direction types.Direction) string {
	letterColor := color.Blue
	bracketColor := color.DarkBlue
	textColor := color.White

	colorize := func(letters string, text string) string {
		return fmt.Sprintf("%s%s%s%s",
			color.Colorize(bracketColor, "["),
			color.Colorize(letterColor, letters),
			color.Colorize(bracketColor, "]"),
			color.Colorize(textColor, text))
	}

	switch direction {
	case types.DirectionNorth:
		return colorize("N", "orth")
	case types.DirectionNorthEast:
		return colorize("NE", "North East")
	case types.DirectionEast:
		return colorize("E", "ast")
	case types.DirectionSouthEast:
		return colorize("SE", "South East")
	case types.DirectionSouth:
		return colorize("S", "outh")
	case types.DirectionSouthWest:
		return colorize("SW", "South West")
	case types.DirectionWest:
		return colorize("W", "est")
	case types.DirectionNorthWest:
		return colorize("NW", "North West")
	case types.DirectionUp:
		return colorize("U", "p")
	case types.DirectionDown:
		return colorize("D", "own")
	case types.DirectionNone:
		return color.Colorize(color.White, "None")
	}

	panic("Unexpected code path")
}

func Paginate(list []string, width, height int) []string {
	itemLength := func(item string) int {
		return len(color.StripColors(item))
	}

	columns := [][]string{}
	widths := []int{}
	totalWidth := 0

	index := 0
	for {
		column := []string{}
		for ; index < (height*(len(columns)+1)) && index < len(list); index++ {
			column = append(column, list[index])
		}

		columnWidth := 0
		for _, item := range column {
			length := itemLength(item)
			if length > columnWidth {
				columnWidth = length
			}
		}
		columnWidth += 2 // Padding between columns

		if (columnWidth + totalWidth) > width {
			// Column doesn't fit, drop it
			index -= len(column)
			break
		}

		totalWidth += columnWidth
		widths = append(widths, columnWidth)
		columns = append(columns, column)

		if index >= len(list) {
			break
		}
	}

	page := ""

	for i := range columns[0] {
		for j := range columns {
			column := columns[j]

			if i < len(column) {
				item := column[i]
				page += item + strings.Repeat(" ", widths[j]-itemLength(item))
			}
		}

		page += "\r\n"
	}

	pages := []string{page}

	if index < len(list) {
		pages = append(pages, Paginate(list[index:], width, height)...)
	}

	return pages
}

func Atois(strings []string) ([]int, error) {
	ints := make([]int, len(strings))
	for i, str := range strings {
		val, err := strconv.Atoi(str)
		if err != nil {
			return ints, err
		}
		ints[i] = val
	}

	return ints, nil
}

func Atoir(str string, min, max int) (int, error) {
	val, err := strconv.Atoi(str)
	if err != nil {
		return val, fmt.Errorf("%v is not a valid number", str)
	}

	if val < min || val > max {
		return val, fmt.Errorf("Value out of range: %v (%v - %v)", val, min, max)
	}

	return val, nil
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func Abs(x int) int {
	if x < 0 {
		x = -x
	}
	return x
}

func Bound(x, lower, upper int) int {
	if x < lower {
		return lower
	}
	if x > upper {
		return upper
	}
	return x
}

func Filter(list []string, pattern string) []string {
	if pattern == "" {
		return list
	}
	filtered := []string{}

	for _, item := range list {
		if FilterItem(item, pattern) {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

func FilterItem(item, pattern string) bool {
	return strings.Contains(strings.ToLower(color.StripColors(item)), strings.ToLower(pattern))
}

func RemoveLast(input string, check string) string {
	if len(check) == 0 {
		return input
	}
	if len(input) > 0 && input[len(input)-1] == check[0] {
		input = input[:len(input)-1]
	}
	return input
}

func RemoveLastChar(input string) string {
	if len(input) > 0 {
		input = input[:len(input)-1]
	}
	return input
}

func RemoveStringArray(input string, check string) string {
	if len(check) == 0 {
		return input
	}

	for _, ch := range check {
		input = strings.Replace(input, string(ch), "", -1)
	}
	return input
}

func RemoveSpecial(input string) string {
	input = RemoveStringArray(input, "=/\\|+-!$#%^&*//()_`~:;\"'[]{}<>,.@")
	return input
}
