# Cockatoo

> Fetch someone's last tweet, hassle free.

## Usage

Make a `GET` request to `https://cockato.corenzan.com/` followed by someone's username on Twitter, optionally suffixed with the desired format - currently supported `html` (default), `txt`, and `json`. Responses are cached for **1 hour**.

### Examples

```
> GET https://cockatoo.corenzan.com/haggen
< Javascript : The Curious Case of Null >= 0 – Camp Vanilla <a href="https://t.co/K0LrdKswKu">https://t.co/K0LrdKswKu</a>
```

```
> GET https://cockatoo.corenzan.com/haggen.txt
< Javascript : The Curious Case of Null >= 0 – Camp Vanilla https://t.co/K0LrdKswKu
```

```
> GET https://cockatoo.corenzan.com/haggen.json
{"status":"Javascript : The Curious Case of Null &gt;= 0 – Camp Vanilla https://t.co/K0LrdKswKu"}
```

## License

The MIT License © 2017 Corenzan
