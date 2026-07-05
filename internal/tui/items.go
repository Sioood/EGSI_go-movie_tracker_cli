package tui

type menuItem struct {
	title       string
	description string
	route       Route
}

func (i menuItem) Title() string {
	return i.title
}

func (i menuItem) Description() string {
	return i.description
}

func (i menuItem) FilterValue() string {
	return i.title + " " + i.description
}

type movieItem struct {
	title  string
	status string
}

func (i movieItem) Title() string {
	return i.title
}

func (i movieItem) Description() string {
	return i.status
}

func (i movieItem) FilterValue() string {
	return i.title + " " + i.status
}
