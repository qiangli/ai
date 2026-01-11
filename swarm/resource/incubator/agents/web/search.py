import http.client
import urllib.parse
import ssl

def search(q): 
    params = urllib.parse.urlencode({'q': q})
    context = ssl._create_unverified_context()
    conn = http.client.HTTPSConnection("html.duckduckgo.com", context=context)
    conn.request("GET", f"/html/?{params}")
    response = conn.getresponse()
    data = response.read()
    conn.close()
    return data

def main(args):
    q = args.get("query", "")
    if not q:
        return {"error": "Query parameter is required"}
    return search(q)

# if __name__ == "__main__":
#     args = {"query": "dublic ca weather today"}
#     resp = main(args)
#     print(resp)