package model

type Bookmark struct {
	Sort string `json:"sort" csv:"sort"`
	Name string `json:"name" csv:"name"`
	Url  string `json:"url" csv:"url"`
	Key  string `json:"-" csv:"-"`
}

type Bookmarks []Bookmark

func (this Bookmarks) Len() int {
	return len(this)
}

func (this Bookmarks) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

func (this Bookmarks) Less(i, j int) bool {
	return this[i].Key < this[j].Key
}
