# Potential Features For The Future

- [ ] make the history length limit configureable
	- [ ] allow via tui to change the configs
	- [ ] allow via cli to change the configs
- [ ] add a clear command to clear rem 
- [ ] add a REM_HISTORY environment variable that you can use to configure the location that rem uses for the folder. (also change history so it's in .config/rem/history instead of ./config/rem/contents)
- [ ] allow rem to configure it's history via a cli --history command as well
- [ ] detect and dont show binary files in the preview
- [x] convert tui to elm architecture then create more tests
- [ ] allow via tui the ability to delete things from history
- [ ] allow pager style commands on the left side bar mode as well in the tui
- [ ] add ci
- [ ] via cli based get via search - finds the first content in the history that matches the regex that we're searching for
	-	 example `rem search 'hello.*world'` returns the first file in history that has a line that matches hello.*world. 
