Pod::Spec.new do |spec|
  spec.name         = 'Gotrust'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/ThePleasurable/go-trustmachine'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS Trustmachine Client'
  spec.source       = { :git => 'https://github.com/ThePleasurable/go-trustmachine.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Gotrust.framework'

	spec.prepare_command = <<-CMD
    curl https://gotruststore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Gotrust.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
