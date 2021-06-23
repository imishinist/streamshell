# streamshell

Send HTTP request to solr stream handler.

```
echo 'search(techproducts, q="from:1800flowers*", fl="from, to", sort="from asc", qt="/export")' | ./streamshell
```


```
Usage of ./streamshell:
  -collection string
    	solr collection (default "techproducts")
  -host string
    	solr host (default "localhost")
  -i	interactive mode
  -port int
    	solr port (default 8983)
```
