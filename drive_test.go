package drive_test

import (
	"fmt"
	"testing"

	drive "github.com/iochen/msgraph-drive"
)

func TestClient_ListChildren(t *testing.T) {
	client, err := drive.NewGraphClient("7ce56c16-d70e-453f-893d-0d8d0878db3b",
		"cd84edce-d529-411f-b9a5-c8b159ca0c1d",
		"3I28hc6NC8-GmJLki18_.t068sp8__t3hp")
	if err != nil {
		t.Error(err)
	}
	drv := client.GetDrive("b!xaErFvCtpUaXFEZHOWXkd4Lea4xSTMlJtRYmPUtWwOHy0WIzEwqSR4bpgPfiO6JC")
	items, err := drv.ListChildren("")
	if err != nil {
		t.Error(err)
	}
	for i := range items {
		fmt.Printf("%#v\n", items[i])
	}
}

func TestDrive_Item(t *testing.T) {
	client, err := drive.NewGraphClient("7ce56c16-d70e-453f-893d-0d8d0878db3b", "cd84edce-d529-411f-b9a5-c8b159ca0c1d", "3I28hc6NC8-GmJLki18_.t068sp8__t3hp")
	if err != nil {
		t.Error(err)
	}
	drv := client.GetDrive("b!xaErFvCtpUaXFEZHOWXkd4Lea4xSTMlJtRYmPUtWwOHy0WIzEwqSR4bpgPfiO6JC")
	item, err := drv.Item("/")
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%#v\n", *item)
}
