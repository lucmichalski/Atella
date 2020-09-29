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

  desc "Read database and processing status"
  task status: :environment do
    settings = Rails.application.config.atella  
    masters = Host.where(:is_master => true)
    redis = Redis.new(host: settings["atella"]["redisHost"])
    redisVectors = Array.new
    statuses = Hash.new
    if masters.nil?
      print "Not enouth masters!"
    end
    masters.each do |m|
      _r = redis.get(m.hostname
      redisVectors.append(r) unless _r.nil?
    end
    redisVectors.each do |v|
      _s = JSON.parse(v)
      s = _s["status"]
      print "#{s}\n"
    end
  end
end
