package infrastructure

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	dseedwork "github.com/ronnieholm/golang-onion-architecture-sample/domain/seedwork"
)

// TODO: should cursorToOffset return error or AssertNoErr? A user could've
// tampered with the values (or bug in client app adding those to URL) so we
// don't want to record it as a server bug. What does F# do?

func CursorToOffset(c *dseedwork.Cursor) (*int64, error) {
	if c != nil {
		bytes, err := base64.StdEncoding.DecodeString(c.Value)
		if err != nil {
			return nil, fmt.Errorf("unable to decode cursor %s: %w", c.Value, err)
		}
		decoded := string(bytes)
		offset, err := strconv.ParseInt(decoded, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("unable to parse cursor %s: %w", decoded, err)
		}
		return &offset, nil

	}
	var defaultOffset int64 = 0
	return &defaultOffset, nil
}

func OffsetsToCursor(pageEndOffset, globalEndOffset int64) *dseedwork.Cursor {
	if pageEndOffset == globalEndOffset {
		return nil
	}
	a := strconv.FormatInt(pageEndOffset, 10)
	b := base64.StdEncoding.EncodeToString([]byte(a))
	c, err := dseedwork.NewCursor(b)
	if err != nil {
		panic(fmt.Errorf("invalid cursor %s: %w", b, err))
	}
	return c
}

type Clock struct{}

func (c Clock) UtcNow() time.Time {
	return time.Now().UTC()
}
