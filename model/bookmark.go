package model

type Bookmark struct {
	Sort string `json:"sort"`
	Name string `json:"name"`
	Url  string `json:"url"`
	Icon string `json:"-"`
	Key  string `json:"-"`
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
