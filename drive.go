package drive

import (
	"fmt"
	"strings"
)

type Drive struct {
	ID     string
	Client *Client
}

func (cli *Client) GetDrive(id string) *Drive {
	return &Drive{
		ID:     id,
		Client: cli,
	}
}

func (drv *Drive) ListChildren(path string) ([]*Item, error) {
	path = strings.Trim(path, "/")
	var source string
	switch path {
	case "root", "":
		source = fmt.Sprintf("/drives/%s/items/root/children", drv.ID)
	default:
		source = fmt.Sprintf("/drives/%s/items/root:/%s:/children", drv.ID, path)
	}
	marsh := &struct {
		Items []*Item `json:"value"`
	}{}
	err := drv.Client.makeGETAPICall(source, nil, marsh)
	if err != nil {
		return nil, err
	}
	return marsh.Items, nil
}

func (drv *Drive) Item(path string) (*Item, error) {
	path = strings.Trim(path, "/")
	var source string
	switch path {
	case "root", "":
		source = fmt.Sprintf("/drives/%s/items/root", drv.ID)
	default:
		source = fmt.Sprintf("/drives/%s/root:/%s", drv.ID, path)
	}
	marsh := &Item{}
	err := drv.Client.makeGETAPICall(source, nil, marsh)
	if err != nil {
		return nil, err
	}
	return marsh, nil
}
