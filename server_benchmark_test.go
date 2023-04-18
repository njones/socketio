package socketio

import (
	"crypto/md5"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"math/rand"
	"strconv"
	"testing"
	"time"

	sios "github.com/njones/socketio/session"
	"github.com/stretchr/testify/assert"
)

func TestSocketIDQuickPrefix(t *testing.T) {
	now := time.Date(1946, time.February, 14, 10, 00, 00, 00, time.UTC) // February 14, 1946 - The date ENIAC was demonstrated to the world
	socketID := sios.ID(_socketIDQuickPrefix(now)() + "ABC123def456")

	assert.Equal(t, sios.ID("f09f838af09f8384f09f839bf09f8387f09f8396::ABC123def456"), socketID)
	assert.Equal(t, "f09f838af09f8384f09f839bf09f8387f09f8396::ABC123def456", socketID.String())
	assert.Equal(t, "f09f838af09f8384f09f839bf09f8387f09f8396::ABC123def456", fmt.Sprintf("%s", socketID))
	assert.Equal(t, "🃊🃄🃛🃇🃖::ABC123def456", socketID.GoString())
	assert.Equal(t, "🃊🃄🃛🃇🃖::ABC123def456", fmt.Sprintf("%#v", socketID))
}

func socketIDQuickPrefix32() string {
	src := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(src)

	cards := [][]rune{
		{127137, 127150}, // spades
		{127153, 127166}, // hearts
		{127169, 127182}, // diamonds
		{127185, 127198}, // clubs
	}

	prefix := make([]rune, 5)
	for i := range prefix {
		suit := rnd.Intn(4)
		card := int32(rnd.Intn(int(cards[suit][1]-cards[suit][0]-1))) + cards[suit][0]
		prefix[i] = card
	}

	enc := base32.HexEncoding.EncodeToString([]byte(string(prefix)))
	return enc + "::"
}

func socketIDQuickPrefix64() string {
	src := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(src)

	cards := [][]rune{
		{127137, 127150}, // spades
		{127153, 127166}, // hearts
		{127169, 127182}, // diamonds
		{127185, 127198}, // clubs
	}

	prefix := make([]rune, 5)
	for i := range prefix {
		suit := rnd.Intn(4)
		card := int32(rnd.Intn(int(cards[suit][1]-cards[suit][0]-1))) + cards[suit][0]
		prefix[i] = card
	}

	enc := base64.StdEncoding.EncodeToString([]byte(string(prefix)))
	return enc + "::"
}

func socketIDQuickPrefixHex() string {
	src := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(src)

	cards := [][]rune{
		{127137, 127150}, // spades
		{127153, 127166}, // hearts
		{127169, 127182}, // diamonds
		{127185, 127198}, // clubs
	}

	prefix := make([]rune, 5)
	for i := range prefix {
		suit := rnd.Intn(4)
		card := int32(rnd.Intn(int(cards[suit][1]-cards[suit][0]-1))) + cards[suit][0]
		prefix[i] = card
	}

	enc := hex.EncodeToString([]byte(string(prefix)))
	return enc + "::"
}

func socketIDQuickPrefixMD5() string {
	src := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(src)

	cards := [][]rune{
		{127137, 127150}, // spades
		{127153, 127166}, // hearts
		{127169, 127182}, // diamonds
		{127185, 127198}, // clubs
	}

	prefix := make([]rune, 5)
	for i := range prefix {
		suit := rnd.Intn(4)
		card := int32(rnd.Intn(int(cards[suit][1]-cards[suit][0]-1))) + cards[suit][0]
		prefix[i] = card
	}

	hsh := md5.Sum([]byte(string(prefix)))
	return string(hsh[:]) + "::"
}

func socketIDQuickPrefixCRC32() string {
	src := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(src)

	cards := [][]rune{
		{127137, 127150}, // spades
		{127153, 127166}, // hearts
		{127169, 127182}, // diamonds
		{127185, 127198}, // clubs
	}

	prefix := make([]rune, 5)
	for i := range prefix {
		suit := rnd.Intn(4)
		card := int32(rnd.Intn(int(cards[suit][1]-cards[suit][0]-1))) + cards[suit][0]
		prefix[i] = card
	}

	i := crc32.ChecksumIEEE([]byte(string(prefix)))
	return strconv.FormatUint(uint64(i), 10) + "::"
}

func BenchmarkSocketIDPrefixEncoding32(b *testing.B) {
	for n := 0; n < b.N; n++ {
		socketIDQuickPrefix32()
	}
}

func BenchmarkSocketIDPrefixEncoding64(b *testing.B) {
	for n := 0; n < b.N; n++ {
		socketIDQuickPrefix64()
	}
}

func BenchmarkSocketIDPrefixEncodingHex(b *testing.B) {
	for n := 0; n < b.N; n++ {
		socketIDQuickPrefixHex()
	}
}

func BenchmarkSocketIDPrefixEncodingMD5(b *testing.B) {
	for n := 0; n < b.N; n++ {
		socketIDQuickPrefixMD5()
	}
}

func BenchmarkSocketIDPrefixEncodingCRC32(b *testing.B) {
	for n := 0; n < b.N; n++ {
		socketIDQuickPrefixMD5()
	}
}
