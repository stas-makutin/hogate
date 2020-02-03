@for /R %%f in (*.opus) do @(
	curl -H "Authorization: OAuth <token>" -H "Content-Type: multipart/form-data" -X POST -F "file=@%%f" "https://dialogs.yandex.net/api/v1/skills/<skill id>/sounds"
)