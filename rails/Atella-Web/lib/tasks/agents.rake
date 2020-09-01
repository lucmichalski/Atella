namespace :agents do
  desc "Read config and update database entry"
  task hosts: :environment do
    Host.reload_hosts_config
    settings = Rails.application.config.atella  
    error = nil
    begin
      mastersConfig = TOML.load_file(settings["atella"]["masterServersConfig"])
      sectorsConfig = TOML.load_file(settings["atella"]["sectorsConfig"])
      securityConfig = TOML.load_file(settings["atella"]["securityConfig"])
    rescue => v
      error = v
    end
    
    if error 
      STDERR.print("Fatal: #{error}\n")  
    end
    hosts = Host.all
    hosts.each do |h|
      next if securityConfig["security"].nil?
      res = h.wrap_version(securityConfig["security"]["code"])
      h.save if res
      print("#{h.address} #{h.hostname}\n")
    end
  end
end
