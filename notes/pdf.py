from pypdf import PdfReader

reader = PdfReader("notatnik-rafala.pdf")

text = ""
for i in range(len(reader.pages)):
    text += reader.pages[i].extract_text()
    text += "\n"

with open("text.txt", 'w') as f:
    f.write(text)