package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

const page = `<!doctype html>
<html>
<body>
<button id="link">Connect RBC</button>
<pre id="out"></pre>
<script src="https://cdn.plaid.com/link/v2/stable/link-initialize.js"></script>
<script>
document.getElementById("link").onclick = async () => {
    const { link_token } = await (await fetch("/link-token")).json();
    Plaid.create({
        token: link_token,
        onSuccess: async (public_token) => {
            const res = await fetch("/exchange", { method: "POST", body: public_token });
            document.getElementById("out").textContent = await res.text();
        },
    }).open();
};
</script>
</body>
</html>`

func plaid(path string, body map[string]any) (map[string]any, error) {
	body["client_id"] = os.Getenv("PLAID_CLIENT_ID")
	body["secret"] = os.Getenv("PLAID_SECRET")
	payload, _ := json.Marshal(body)
	res, err := http.Post("https://production.plaid.com"+path, "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("%s: %v", path, out)
	}
	return out, nil
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, page)
	})
	http.HandleFunc("/link-token", func(w http.ResponseWriter, r *http.Request) {
		out, err := plaid("/link/token/create", map[string]any{
			"user":          map[string]string{"client_user_id": "franco"},
			"client_name":   "personal-workflows",
			"products":      []string{"transactions"},
			"country_codes": []string{"CA"},
			"language":      "en",
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(out)
	})
	http.HandleFunc("/exchange", func(w http.ResponseWriter, r *http.Request) {
		publicToken, _ := io.ReadAll(r.Body)
		out, err := plaid("/item/public_token/exchange", map[string]any{"public_token": string(publicToken)})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		fmt.Fprintf(w, "access_token=%s\nitem_id=%s\n", out["access_token"], out["item_id"])
		log.Printf("access_token=%s item_id=%s", out["access_token"], out["item_id"])
	})
	log.Println("open http://localhost:8411")
	log.Fatal(http.ListenAndServe("localhost:8411", nil))
}
