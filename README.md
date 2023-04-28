# Havoc Profile Generator by 0xv1n

Purpose: Simple GUI tool to generate a Havoc C2 profile for Demon.

![](mainwindow.png)

## Usage

1. `go run main.go`
2. fill in parameters as desired
3. click save
4. profile is saved as `profile.yaotl` if no name is specified.

### Field Validation/Error Handling
- The form does not currently validate that you enter a correct type
- If you do not specify a value for a field defined as `Required` in the docs, the profile you save will auto-populate that field with a placeholder `"REQUIRED_FIELD"`.

## TODO

- [x] Allow users to specify file name
- [ ] Parse existing `yaotl` profile to allow GUI based modification of existing profiles.
- [x] Maybe add Monaco font to keep theming consistent with the Havoc client.
