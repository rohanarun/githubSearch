## Explanation
This is a web app that pulls source code from GitHub to run search functions. The backend works using the Golang, the Github REST API, and ElasticSearch. This allows developers to search even new repos, and search generally within any code or language on Github. The 'References' function works in all languages, but the 'Go-To-Def' function only works in javascript as a PoC due to time constraints. The frontend relays the user input to the backend go script using websockets, and parses the search results from ElasticSearch into a usable interface. Enter a Github repo in the format 'owner/repo,' and choose a file from the explorer on the left to view it in the code window. Then select text and right click to run the search functions.

## Building and Running

1) Install the dependencies. Below are instructions for Ubuntu 16.04:
###install Go
sudo apt-get update
sudo apt-get -y upgrade
sudo curl -O https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz
sudo tar -xvf go1.8.linux-amd64.tar.gz
sudo mv go /usr/locall

###install Gin
go get github.com/gin-gonic/gin

###install melody v1
go get github.com/gin-gonic/gin

###install ElasticSearch v5.x
wget -qO - https://artifacts.elastic.co/GPG-KEY-elasticsearch | sudo apt-key add -
sudo apt-get install apt-transport-https
echo "deb https://artifacts.elastic.co/packages/5.x/apt stable main" | sudo tee -a /etc/apt/sources.list.d/elastic-5.x.list
sudo apt-get update && sudo apt-get install elasticsearch

###Install Elasticsearch go client v5
go get gopkg.in/olivere/elastic.v5


2) Navigate to the code folder and run the go script:

go run analyze.go 

3) If you setup Nginx reverse proxy, you can go to http://localhost. Otherwise, go to http://localhost:5000 to test the app.

4) You can pull code from any repo, but smaller ones work faster. For example try "jessabean/rgb-hex-converter"

5) After you enter the repo name, it will populate the file explorer. Clicking on the file names will display the code in the code window.

6) Highlight text using the mouse, and right click to access the search functions. You can run Go-to-def or References. The output will display on the bottom as the "hover tool tips." In this case they don't actually hover over the code due to time constraints, and instead display in a list on the bottom. You will also see line numbers and the time it took to search.

## Testing
1) Enter "jessabean/rgb-hex-converter" into the text input.
2) Click js/rgb-hex.js in the left menu.
3) Highlight "convertToHex." Right click and select "Show All References."
4) It should show both the implementation and declaration in the bottom window. 
5) Now again select "convertToHex." Right click and select "Jump to Definition."
6) This time it should only show the declaration in the bottom window. 
7) You can try loading larger repos like "facebook/react," but it will take longer to index. If you do load facebook/react, you can try the same steps 2-6 with "scripts/rollup/sync.js" and the "doSync" function to see the output below.

"Search Results:
Search Time: 1 milliseconds
File Name: scripts/rollup/sync.js 
13: function doSync(buildPath, destPath) {"

