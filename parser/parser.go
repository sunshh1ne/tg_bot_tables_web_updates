package parser

import (
	"fmt"
	"golang.org/x/net/html"
	"net/http"
	"strings"
)

func getNodeID(n *html.Node) string {
	for _, attr := range n.Attr {
		if attr.Key == "id" {
			return attr.Val
		}
	}
	return ""
}

func getNodeClass(n *html.Node, class string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			classes := strings.Fields(attr.Val)
			for _, c := range classes {
				if c == class {
					return true
				}
			}
		}
	}
	return false
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
	var processAllProduct func(*html.Node, bool)
	processAllProduct = func(n *html.Node, flag bool) {
		if getNodeID(n) == "grid-bottom-bar" {
			return
		}
		if flag && n.Type == html.TextNode {
			if len(ret) > 0 && ret[len(ret)-1] != '\n' {
				ret += "\t"
			}
			ret += n.Data
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			processAllProduct(c, flag || (n.Type == html.ElementNode && n.Data == "tr"))
		}
		if n.Type == html.ElementNode && n.Data == "tr" {
			ret += "\n"
		}
	}
	processAllProduct(doc, false)
	return ret, nil
}
