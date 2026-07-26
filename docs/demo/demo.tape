Output docs/demo/demo.gif

Require java
Require mvn
Require jup

Set Shell "bash"

Set FontFamily "LiterationMono Nerd Font Mono"
Set FontSize 20
Set Width 1200
Set Height 720
Set Padding 20
Set Margin 0
Set BorderRadius 12

Set Theme "Raycast_Dark"
Set WindowBar RingsRight
Set CursorBlink false

Hide
Type `repo="$PWD"`
Enter
Type `export JAVAUP_HOME="$repo/.vhs/javaup"`
Enter
Type `export PATH="$JAVA_HOME/bin:$PATH"`
Enter
Type `rm -rf -- "$JAVAUP_HOME"`
Enter
Type "export PS1='jup \W > '"
Enter
Type "clear"
Enter
Show

Type "jup version"
Enter
Sleep 1s

Type "java -version"
Enter
Sleep 1s

Type "cd docs/demo/java8-app"
Enter
Type "jup init"
Enter
Sleep 1s

Type "cd ../java17-app"
Enter
Type "jup init"
Enter
Sleep 1s

Type `cd ~`
Enter
Type "jup run mvn clean compile"
Enter
Sleep 1s
Enter
Sleep 2s

Type "jup run mvn clean compile --project java8"
Enter
Sleep 2s

Type "java -version"
Enter
Sleep 2s
