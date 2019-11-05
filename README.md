# Peerster
## How to run the code
- goLang
	- install the requirements of packages
		```
		go install
		```
	
	- Compile the code
		```
		go build
		```
	
	- Run the code
	```
	./Peerster 
	```
	
	- Please refer to the handout for the further flag information
- Run Gui: two flags added
	- ```-gui```: enable the GUI
	- ```-GUIPort=8081```: set the port, by default it should be ```8080``` in the localhost


- Vue Framework (You need to compile it by yourself, as the compiled files will be too large for the submission, really sorry about this)
	- go to ``` web server/gui ```
	
	- Dependency Installation
	``` npm i ``` to generate ```node_modules```
	
	- Production env
		``` npm run build ``` and generate folder ```dist/```
		
	- We set the port as ``` 127.0.0.1:8080 ``` by default

- For the homework2, I have prepared a test script, called ```arrow_test.sh```. Enjoy it.	