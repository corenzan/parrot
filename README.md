# Parrot

> Fetch someone's last tweet, hassle free.

## Usage

Make a `GET` request to `https://parrot.corenzan.com/` followed by someone's username on Twitter, optionally suffixed with the desired format - currently supported `html` (default), `txt`, and `json`. Also, please note:

- Responses are cached for 1 hour.
- The HTML format provide anchors for URLs found in the tweet.

### Examples

```
GET https://parrot.corenzan.com/haggen

Javascript : The Curious Case of Null >= 0 – Camp Vanilla <a href="https://t.co/K0LrdKswKu">https://t.co/K0LrdKswKu</a>
```

```
GET https://parrot.corenzan.com/haggen.txt

Javascript : The Curious Case of Null >= 0 – Camp Vanilla https://t.co/K0LrdKswKu
```

```
GET https://parrot.corenzan.com/haggen.json

{
 "status": "Javascript : The Curious Case of Null &gt;= 0 – Camp Vanilla https://t.co/K0LrdKswKu"
}
```

## License

The MIT License © 2017 Corenzan
