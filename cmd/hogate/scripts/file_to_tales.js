function puts(str) {
	WScript.Echo(str)
}

function print(str) {
	WScript.StdOut.Write(str)
}

function quit(code) {
	WScript.Quit(code)
}

// ---------------------------------------------------------------------------

var inputFile = ""

for (var i = 0; i < WScript.Arguments.length; i++)
{
	var arg = WScript.Arguments(i)
	if (inputFile.length <= 0)
		inputFile = arg;
}

var fso = new ActiveXObject("Scripting.FileSystemObject")
if (!fso)
{
	puts("Cannot create Scripting.FileSystemObject object.")
	quit(1);
}
var file = fso.OpenTextFile(inputFile, 1)
if (!file)
{
	puts("Cannot open file " + inputFile)
	quit(1)
}
eval('var json = ' + file.ReadAll())
file.Close()
if (!(json !== null && typeof json === 'object' && 'sounds' in json && json.sounds instanceof Array)) {
	puts('Unexpected file format')
	quit(1)
}

var tales = {}
for (var i = 0; i < json.sounds.length; i++) {
	var item = json.sounds[i]
	if (!(item !== null && typeof item === 'object' && 'id' in item && 'originalName' in item 
		&& item.originalName !== null && typeof item.originalName === 'string' && item.originalName.length > 0
		&& item.id !== null && typeof item.id === 'string' && item.id.length > 0
	)) {
		continue
	}
	var id = item.id
	var name = item.originalName

	var p = name.lastIndexOf('.')
	if (p <= 0) {
		puts('kore');
		continue
	}
	name = name.substring(0, p)
	var p = name.lastIndexOf('_')
	if (p <= 0) {
		puts('sore');
		continue
	}
	ordinal = parseInt(name.substring(p + 1))
	name = name.substring(0, p)
	if (name.length <= 0 || isNaN(ordinal) || ordinal < 0) {
		continue;
	}
	
	if (name in tales) {
		tales[name].push({"ordinal": ordinal, "id": id})
	} else {
		tales[name] = [{"ordinal": ordinal, "id": id}]
	}
}

for (name in tales) {
	parts = tales[name]
	parts.sort(function (a, b) { return a.ordinal < b.ordinal ? -1: a.ordinal > b.ordinal ? 1 : 0; })
	puts('- name: ' + name)
	puts('  type: story')
	puts('  length: ' + parts.length * 120)
	puts('  parts:')
	for (var i = 0; i < parts.length; i++) {
		puts('    - ' + parts[i].id)
	}
	puts('')
}

quit(0)