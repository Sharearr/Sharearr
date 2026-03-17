package main

import "mime/multipart"

type IDParam struct {
	ID int64 `uri:"id"`
}

type CategoryParam struct {
	CategoryIDs []int64 `form:"cat" collection_format:"csv"`
}

type TorznabQuery struct {
	Q        string `form:"q"`
	Type     string `form:"t"`
	Cat      []int  `form:"cat" collection_format:"csv"`
	Extended bool   `form:"extended"`
	Limit    int    `form:"limit"  binding:"gte=1,lte=100"`
	Offset   int    `form:"offset" binding:"gte=0"`
	*TorznabTvQuery
}

type TorznabTvQuery struct {
	Season  string `form:"season"`
	Episode string `form:"ep"`
}

type CreateTorrent struct {
	CategoryParam
	File *multipart.FileHeader `form:"file" binding:"required"`
}
