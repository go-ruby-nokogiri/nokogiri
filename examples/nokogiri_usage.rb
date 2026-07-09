# frozen_string_literal: true

require "nokogiri"

# Parse an HTML document.
html = <<~HTML
  <html><body>
    <h1 id="title">Fruit list</h1>
    <ul class="fruits">
      <li class="fruit">Apple</li>
      <li class="fruit">Banana</li>
      <li class="fruit">Cherry</li>
    </ul>
    <a href="https://example.com">Home</a>
  </body></html>
HTML
doc = Nokogiri::HTML(html)

# at_css returns the first match; #[] reads an attribute; #text reads content.
heading = doc.at_css("h1")
puts "#{heading.text} (id=#{heading["id"]})"   # => Fruit list (id=title)
puts "link -> #{doc.at_css("a")["href"]}"      # => link -> https://example.com

# css returns a NodeSet you can iterate and map over.
fruits = doc.css("li.fruit")
puts "#{fruits.length} fruits: #{fruits.map { |n| n.text }.join(", ")}"

# Parse XML and query it with XPath.
xml = Nokogiri::XML("<catalog><book id='1'>Go</book><book id='2'>Ruby</book></catalog>")
puts "books: #{xml.xpath("//book").length}"
puts "book 2: #{xml.at_xpath("//book[@id='2']").text}"
xml.css("book").each { |b| puts "  #{b["id"]}: #{b.text}" }
