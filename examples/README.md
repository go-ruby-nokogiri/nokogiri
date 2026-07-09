# nokogiri examples

Runnable pure-Ruby usage of the `nokogiri` HTML/XML parser, verified under the [rbgo](https://github.com/go-embedded-ruby) interpreter.

```sh
rbgo examples/nokogiri_usage.rb
```

| File | Shows |
| --- | --- |
| `nokogiri_usage.rb` | Parse HTML with `Nokogiri::HTML`, query with `#css` / `#at_css`, read text and attributes, then parse XML with `Nokogiri::XML` and query it with `#xpath` / `#at_xpath`. |
