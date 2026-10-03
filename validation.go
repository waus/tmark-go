package tmark

import (
	"fmt"
	"regexp"
)

var canonicalNumber = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]*[1-9])?$`)

func validNumber(s string) bool { return s != "-0" && canonicalNumber.MatchString(s) }

func validateNode(value any) error {
	switch node := value.(type) {
	case Header:
		if node.Size < 1 || node.Size > 6 {
			return fmt.Errorf("tmark: header size must be 1..6")
		}
	case Video:
		if node.Preview == "" {
			return fmt.Errorf("tmark: video preview is required")
		}
	case ListItem:
		if node.Type != nil {
			switch *node.Type {
			case "a", "A", "i", "I", "1", "checkbox":
			default:
				return fmt.Errorf("tmark: invalid list item type %q", *node.Type)
			}
		}
	case Cell:
		if node.Align != nil {
			switch *node.Align {
			case TableCellAlignLeft, TableCellAlignCenter, TableCellAlignRight:
			default:
				return fmt.Errorf("tmark: invalid cell align %q", *node.Align)
			}
		}
		if node.Valign != nil {
			switch *node.Valign {
			case TableCellValignTop, TableCellValignMiddle, TableCellValignBottom:
			default:
				return fmt.Errorf("tmark: invalid cell valign %q", *node.Valign)
			}
		}
	}
	return nil
}
