package parser

import (
	"fmt"
	"golang.org/x/net/html"
	"log"
	"net/http"
	"strconv"
)

func getNodeID(n *html.Node) string {
	for _, attr := range n.Attr {
		if attr.Key == "id" {
			return attr.Val
		}
	}
	return ""
}

func ParseSite(url string) (string, error) {
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	doc, err := html.Parse(response.Body)
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}

	ret := ""
	counter := 0
	var processAllProduct func(*html.Node, bool)
	processAllProduct = func(n *html.Node, flag bool) {
		if getNodeID(n) == "grid-bottom-bar" || (n.Type == html.ElementNode && n.Data == "th") {

			return
		}
		if flag && n.Type == html.TextNode {
			ret += string('\x01') + strconv.Itoa(counter) + string('\x01')
			counter++
			ret += n.Data
			return
		}
		ret_len := len(ret)
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			processAllProduct(c, flag || (n.Type == html.ElementNode && n.Data == "tr"))
		}
		if flag && n.Type == html.ElementNode && n.Data == "td" && ret_len == len(ret) {
			ret += string('\x01') + strconv.Itoa(counter) + string('\x01')
			counter++
			return
		}
		if n.Type == html.ElementNode && n.Data == "tr" {
			ret += string('\x01') + "\n" + string('\x01')
			counter = 0
		}
	}
	processAllProduct(doc, false)
	return ret, nil
}

func checkRange(ranges string) bool {
	var list []uint8
	for i := 0; i < len(ranges); i++ {
		if ranges[i] < '0' || ranges[i] > '9' {
			list = append(list, ranges[i])
		}
	}

	if len(list) != 3 {
		return false
	}

	for i := 0; i < len(list); i++ {
		if list[i] != ':' && list[i] != '-' {
			return false
		}
	}

	return list[0] == ':' && list[1] == '-' && list[2] == ':'
}

func rangeParse(ranges string) (int, string, int, string) {
	x1, x2 := 0, 0
	y1, y2 := "", ""

	i := 0
	for i < len(ranges) {
		if ranges[i] >= '0' && ranges[i] <= '9' {
			x1 = x1*10 + (int)(ranges[i]-'0')
		} else {
			break
		}
		i++
	}
	i++
	for i < len(ranges) {
		if ranges[i] >= '0' && ranges[i] <= '9' {
			y1 += string(ranges[i])
		} else {
			break
		}
		i++
	}
	i++

	for i < len(ranges) {
		if ranges[i] >= '0' && ranges[i] <= '9' {
			x2 = x2*10 + (int)(ranges[i]-'0')
		} else {
			break
		}
		i++
	}
	i++
	for i < len(ranges) {
		if ranges[i] >= '0' && ranges[i] <= '9' {
			y2 += string(ranges[i])
		} else {
			break
		}
		i++
	}
	return x1, y1, x2, y2
}

func nextIndex(data string, pos int) int {
	for pos < len(data) {
		if data[pos] == '\x01' {
			return pos
		}
		pos++
	}
	return -1
}

func getText(data string, pos int) string {
	if data[pos] == '\x01' {
		return ""
	}
	nextpos := nextIndex(data, pos)
	text := data[pos:nextpos]
	return text
}

func GetDifferences(data1, data2, ranges string) ([]string, []string) {
	if !checkRange(ranges) {
		return nil, nil
	}

	x1, y1, x2, y2 := rangeParse(ranges)

	l, r := 0, 0
	x := 0

	var before []string
	var after []string
	for l < len(data1) && r < len(data2) {
		if data1[l] != '\x01' {
			log.Fatal(l, data1[l])
		}
		pos_data1 := nextIndex(data1, l+1)
		if pos_data1 == -1 {
			fmt.Println("1", l, data1[l], len(data1))
		}
		y_data := data1[l+1 : pos_data1]
		pos_data2 := nextIndex(data2, r+1)
		if pos_data2 == -1 {
			fmt.Println("2", r, data2[r], len(data2))
		}

		if x2 < x {
			break
		}

		if y_data == "\n" {
			x++
			if pos_data1+1 < len(data1) {
				l = pos_data1 + 1
			} else {
				break
			}
			if pos_data2+1 < len(data2) {
				r = pos_data2 + 1
			} else {
				break
			}
		}

		l = pos_data1 + 1
		r = pos_data2 + 1

		if data1[l] == '\x01' && data2[r] == '\x01' {
			continue
		}

		y_val, err := strconv.Atoi(y_data)
		if err != nil {
			log.Fatal(err)
		}
		y1_val, err := strconv.Atoi(y1)
		if err != nil {
			log.Fatal(err)
		}
		y2_val, err := strconv.Atoi(y2)
		if err != nil {
			log.Fatal(err)
		}
		if !(y1_val > y_val || y2_val < y_val || x1 > x) {
			text1, text2 := getText(data1, l), getText(data2, r)
			if text1 != text2 {
				before = append(before, "Ячейка "+strconv.Itoa(x)+":"+y_data+" Значение: "+text1)
				after = append(after, "Ячейка "+strconv.Itoa(x)+":"+y_data+" Значение: "+text2)
			}
		}

		l = nextIndex(data1, l)
		r = nextIndex(data2, r)
	}
	return before, after
}
