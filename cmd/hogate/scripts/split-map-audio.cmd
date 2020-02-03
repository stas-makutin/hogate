@setlocal ENABLEDELAYEDEXPANSION
@set /a ordinal=0
for /f "delims=" %%a in (%2) do @(
	@set paddedOrdinal=00!ordinal!
	ffmpeg.exe -i %1 %%a -ac 1 -c:a libopus %~n1_!paddedOrdinal:~-3!.opus
	@set /a ordinal=ordinal+1
)
@endlocal
