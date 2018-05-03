package relay_interface

// Based off relay.ConnectionFromArray
// using And interface to get total Count an Page

import (
	"encoding/base64"
	"fmt"
	"github.com/graphql-go/relay"
	"strconv"
	"strings"
	"errors"
)

const PREFIX = "relayxconnection:"

type ArraySliceMetaInfo struct {
	SliceStart  int64 `json:"sliceStart"`
	ArrayLength int64 `json:"arrayLength"`
}

type ConnectionInterface interface {
	TotalCount() (int64, error)
	FetchPage( /* lower */ int64 /* upper */, int64) ([]interface{}, error)
}

func ConnectionFromInterface(
	conn ConnectionInterface,
	args relay.ConnectionArguments,
) (relayConn *relay.Connection, err error) {

	relayConn = &relay.Connection{}
	count, err := conn.TotalCount()
	if err != nil {
		return
	}
	meta := ArraySliceMetaInfo{0, count}
	meta.ArrayLength = count

	sliceEnd := meta.SliceStart + meta.ArrayLength
	beforeOffset := GetOffsetWithDefault(args.Before, meta.ArrayLength)
	afterOffset := GetOffsetWithDefault(args.After, -1)

	startOffset := ternaryMax(meta.SliceStart-1, afterOffset, -1) + 1
	endOffset := ternaryMin(sliceEnd, beforeOffset, meta.ArrayLength)

	if args.First != -1 {
		endOffset = min(endOffset, startOffset+int64(args.First))
	}

	if args.Last != -1 {
		startOffset = max(startOffset, endOffset-int64(args.Last))
	}

	begin := max(startOffset-meta.SliceStart, 0)
	end := meta.ArrayLength - (sliceEnd - endOffset)

	if begin > end {
		relayConn = relay.NewConnection()
		return
	}

	slice, err := conn.FetchPage(begin, end)
	if err != nil {
		return
	}
	var edges = make([]*relay.Edge, len(slice))
	for index, value := range slice {
		edges = append(edges, &relay.Edge{
			Cursor: OffsetToCursor(startOffset + int64(index)),
			Node:   value,
		})
	}

	var firstEdgeCursor, lastEdgeCursor relay.ConnectionCursor
	if len(edges) > 0 {
		firstEdgeCursor = edges[0].Cursor
		lastEdgeCursor = edges[len(edges)-1:][0].Cursor
	}

	lowerBound := int64(0)
	if len(args.After) > 0 {
		lowerBound = afterOffset + 1
	}

	upperBound := meta.ArrayLength
	if len(args.Before) > 0 {
		upperBound = beforeOffset
	}

	hasPreviousPage := false
	if args.Last != -1 {
		hasPreviousPage = startOffset > lowerBound
	}

	hasNextPage := false
	if args.First != -1 {
		hasNextPage = endOffset < upperBound
	}

	relayConn = relay.NewConnection()
	relayConn.Edges = edges
	relayConn.PageInfo = relay.PageInfo{
		StartCursor:     firstEdgeCursor,
		EndCursor:       lastEdgeCursor,
		HasPreviousPage: hasPreviousPage,
		HasNextPage:     hasNextPage,
	}
	return
}

// Creates the cursor string from an offset
func OffsetToCursor(offset int64) relay.ConnectionCursor {
	str := fmt.Sprintf("%v%v", PREFIX, offset)
	return relay.ConnectionCursor(base64.StdEncoding.EncodeToString([]byte(str)))
}

func CursorToOffset(cursor relay.ConnectionCursor) (int64, error) {
	str := ""
	b, err := base64.StdEncoding.DecodeString(string(cursor))
	if err == nil {
		str = string(b)
	}
	str = strings.Replace(str, PREFIX, "", -1)
	offset, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, errors.New("invalid cursor")
	}
	return offset, nil
}

func GetOffsetWithDefault(cursor relay.ConnectionCursor, defaultOffset int64) int64 {
	if cursor == "" {
		return defaultOffset
	}
	offset, err := CursorToOffset(cursor)
	if err != nil {
		return defaultOffset
	}
	return offset
}

func max(a, b int64) int64 {
	if a < b {
		return b
	}
	return a
}

func ternaryMax(a, b, c int64) int64 {
	return max(max(a, b), c)
}

func min(a, b int64) int64 {
	if a > b {
		return b
	}
	return a
}

func ternaryMin(a, b, c int64) int64 {
	return min(min(a, b), c)
}
