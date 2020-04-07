@for /R %%f in (*.mp4) do @(
	D:\data\software\multimedia\ffmpeg\x64\bin\ffmpeg.exe -i %%f -q:a 0 -map a %%~nf.mp3
)