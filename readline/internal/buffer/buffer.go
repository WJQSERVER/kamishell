package buffer

import (
	"github.com/mattn/go-runewidth"
	"unicode"
)

type Buffer struct {
	data   []rune
	cursor int // character position
}

func NewBuffer() *Buffer {
	return &Buffer{
		data: make([]rune, 0),
	}
}

func (b *Buffer) Insert(r rune) {
	b.data = append(b.data, 0)
	copy(b.data[b.cursor+1:], b.data[b.cursor:])
	b.data[b.cursor] = r
	b.cursor++
}

func (b *Buffer) Delete() {
	if b.cursor < len(b.data) {
		b.data = append(b.data[:b.cursor], b.data[b.cursor+1:]...)
	}
}

func (b *Buffer) Backspace() {
	if b.cursor > 0 {
		b.cursor--
		b.data = append(b.data[:b.cursor], b.data[b.cursor+1:]...)
	}
}

func (b *Buffer) MoveLeft() {
	if b.cursor > 0 {
		b.cursor--
	}
}

func (b *Buffer) MoveRight() {
	if b.cursor < len(b.data) {
		b.cursor++
	}
}

func (b *Buffer) MoveWordLeft() {
	if b.cursor == 0 {
		return
	}

	i := b.cursor
	// Skip spaces to the left
	for i > 0 && unicode.IsSpace(b.data[i-1]) {
		i--
	}
	// Skip non-spaces to the left
	for i > 0 && !unicode.IsSpace(b.data[i-1]) {
		i--
	}
	b.cursor = i
}

func (b *Buffer) MoveWordRight() {
	if b.cursor == len(b.data) {
		return
	}

	i := b.cursor
	// Skip spaces to the right
	for i < len(b.data) && unicode.IsSpace(b.data[i]) {
		i++
	}
	// Skip non-spaces to the right
	for i < len(b.data) && !unicode.IsSpace(b.data[i]) {
		i++
	}
	b.cursor = i
}

func (b *Buffer) MoveHome() {
	b.cursor = 0
}

func (b *Buffer) MoveEnd() {
	b.cursor = len(b.data)
}

func (b *Buffer) KillToEnd() string {
	killed := string(b.data[b.cursor:])
	b.data = b.data[:b.cursor]
	return killed
}

func (b *Buffer) KillToStart() string {
	killed := string(b.data[:b.cursor])
	b.data = b.data[b.cursor:]
	b.cursor = 0
	return killed
}

func (b *Buffer) String() string {
	return string(b.data)
}

func (b *Buffer) Runes() []rune {
	return b.data
}

func (b *Buffer) Cursor() int {
	return b.cursor
}

func (b *Buffer) SetContent(s string) {
	b.data = []rune(s)
	b.cursor = len(b.data)
}

func (b *Buffer) Clear() {
	b.data = b.data[:0]
	b.cursor = 0
}

// DisplayWidth returns the visual width of the buffer up to a certain point
func (b *Buffer) DisplayWidth(limit int) int {
	if limit > len(b.data) {
		limit = len(b.data)
	}
	return runewidth.StringWidth(string(b.data[:limit]))
}

func (b *Buffer) FullWidth() int {
	return runewidth.StringWidth(string(b.data))
}

func (b *Buffer) DeleteWord() string {
	if b.cursor == len(b.data) {
		return ""
	}

	i := b.cursor
	for i < len(b.data) && unicode.IsSpace(b.data[i]) {
		i++
	}
	for i < len(b.data) && !unicode.IsSpace(b.data[i]) {
		i++
	}

	killed := string(b.data[b.cursor:i])
	b.data = append(b.data[:b.cursor], b.data[i:]...)
	return killed
}

func (b *Buffer) BackspaceWord() string {
	if b.cursor == 0 {
		return ""
	}

	i := b.cursor
	for i > 0 && unicode.IsSpace(b.data[i-1]) {
		i--
	}
	for i > 0 && !unicode.IsSpace(b.data[i-1]) {
		i--
	}

	killed := string(b.data[i:b.cursor])
	b.data = append(b.data[:i], b.data[b.cursor:]...)
	b.cursor = i
	return killed
}

func (b *Buffer) TransposeChars() {
	if len(b.data) < 2 || b.cursor == 0 {
		return
	}
	if b.cursor == len(b.data) {
		b.data[b.cursor-2], b.data[b.cursor-1] = b.data[b.cursor-1], b.data[b.cursor-2]
	} else {
		b.data[b.cursor-1], b.data[b.cursor] = b.data[b.cursor], b.data[b.cursor-1]
		b.cursor++
	}
}

func (b *Buffer) TransposeWords() {
	if len(b.data) == 0 {
		return
	}

	cursor := b.cursor
	for cursor > 0 && unicode.IsSpace(b.data[cursor-1]) {
		cursor--
	}

	word2End := cursor
	for word2End < len(b.data) && !unicode.IsSpace(b.data[word2End]) {
		word2End++
	}
	word2Start := cursor
	for word2Start > 0 && !unicode.IsSpace(b.data[word2Start-1]) {
		word2Start--
	}

	if word2Start == word2End {
		return
	}

	word1End := word2Start
	for word1End > 0 && unicode.IsSpace(b.data[word1End-1]) {
		word1End--
	}
	word1Start := word1End
	for word1Start > 0 && !unicode.IsSpace(b.data[word1Start-1]) {
		word1Start--
	}

	if word1Start == word1End {
		return
	}

	word1 := append([]rune(nil), b.data[word1Start:word1End]...)
	word2 := append([]rune(nil), b.data[word2Start:word2End]...)

	newData := make([]rune, 0, len(b.data))
	newData = append(newData, b.data[:word1Start]...)
	newData = append(newData, word2...)
	newData = append(newData, b.data[word1End:word2Start]...)
	newData = append(newData, word1...)
	newData = append(newData, b.data[word2End:]...)

	b.data = newData
	b.cursor = word2End
}

func (b *Buffer) CapitalizeWord() {
	if b.cursor >= len(b.data) {
		return
	}

	i := b.cursor
	for i < len(b.data) && unicode.IsSpace(b.data[i]) {
		i++
	}

	if i < len(b.data) && unicode.IsLower(b.data[i]) {
		b.data[i] = unicode.ToUpper(b.data[i])
	}

	for i < len(b.data) && !unicode.IsSpace(b.data[i]) {
		i++
		if i < len(b.data) && unicode.IsUpper(b.data[i]) {
			b.data[i] = unicode.ToLower(b.data[i])
		}
	}

	b.cursor = i
}

func (b *Buffer) UppercaseWord() {
	if b.cursor >= len(b.data) {
		return
	}

	i := b.cursor
	for i < len(b.data) && unicode.IsSpace(b.data[i]) {
		i++
	}

	for i < len(b.data) && !unicode.IsSpace(b.data[i]) {
		b.data[i] = unicode.ToUpper(b.data[i])
		i++
	}

	b.cursor = i
}

func (b *Buffer) LowercaseWord() {
	if b.cursor >= len(b.data) {
		return
	}

	i := b.cursor
	for i < len(b.data) && unicode.IsSpace(b.data[i]) {
		i++
	}

	for i < len(b.data) && !unicode.IsSpace(b.data[i]) {
		b.data[i] = unicode.ToLower(b.data[i])
		i++
	}

	b.cursor = i
}
