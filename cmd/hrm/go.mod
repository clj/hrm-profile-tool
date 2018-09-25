module github.com/clj/hrm-profile-tool/cmd/hrm

require (
	github.com/clj/hrm-profile-tool/profile v0.0.0
	github.com/clj/hrm-profile-tool/utils/text v0.0.0
	github.com/clj/hrm-profile-tool/utils/seekbufio v0.0.0

	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/mitchellh/go-homedir v1.0.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.2 // indirect
)

replace github.com/clj/hrm-profile-tool/profile => ../../profile

replace github.com/clj/hrm-profile-tool/utils/text => ../../utils/text

replace github.com/clj/hrm-profile-tool/utils/seekbufio => ../../utils/seekbufio

replace github.com/clj/hrm-profile-tool/instructions => ../../instructions
