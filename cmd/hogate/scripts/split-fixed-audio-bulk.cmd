@for /R %%f in (*.mp3,*.wav) do @(
	ffmpeg.exe -i %%f -map 0 -f segment -segment_time 120 -ac 1 -c:a libopus %%~nf_%%03d.opus
)