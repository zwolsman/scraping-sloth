package jumbo

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
)

const (
	url                = "https://www.jumbo.com/api/graphql"
	pageSize           = 24
	topic              = "jobs-jumbo"
	subscription       = "jobs-jumbo-subscription"
	productQueryFormat = `{"query":"query SearchProducts($input: ProductSearchInput!) {\n\tsearchProducts(input: $input) {\n\t\tstart\n\t\tcount\n\t\tproducts {\n\t\t\tbrand\n\t\t\tean\n\t\t\ttitle\n\t\t\tprices: price {\n\t\t\t\tprice\n\t\t\t\tpromoPrice\n\t\t\t}\n\t\t}\n\t\t__typename\n\t}\n}\n","operationName":"SearchProducts","variables":{"input":{"searchType":"category","searchTerms":"producten","offSet":%d,"currentUrl":"https://www.jumbo.com/producten/","previousUrl":"https://www.jumbo.com/producten/"}}}`
	countQuery         = `{"query":"query SearchProducts($input: ProductSearchInput!) {\n\tsearchProducts(input: $input) {\n\t\tcount\n\t\t__typename\n\t}\n}\n","operationName":"SearchProducts","variables":{"input":{"searchType":"category","searchTerms":"producten","offSet":0,"currentUrl":"https://www.jumbo.com/producten/","previousUrl":"https://www.jumbo.com/producten/"}}}`
)

func createRequest(ctx context.Context, offset int, payload []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	offset -= pageSize
	if offset < 0 {
		offset = 0
	}

	req.Header.Add("Origin", "https://www.jumbo.com")
	req.Header.Add("Referer", fmt.Sprintf("https://www.jumbo.com/producten/?offSet=%d", offset))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Accept-Language", "nl")
	req.Header.Add("User-Agent", "insomnia/2023.2.2")
	return req, nil
}
