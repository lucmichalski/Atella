namespace :agents do
  desc "Read config and update database entry"
  task hosts: :environment do
    Host.reload_hosts_config
    settings = Rails.application.config.atella  
    error = nil
    begin
      securityConfig = TOML.load_file(settings["atella"]["securityConfig"])
    rescue => v
      error = v
    end
    
    if error 
      STDERR.print("[FATAL]: #{error}\n")
      return
    end
    hosts = Host.all
    hosts.each do |h|
      res = h.wrap_version(securityConfig["security"]["code"])
      h.save if res
      STDERR.print("[INFO]: #{h.address} #{h.hostname}\n")
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
      STDERR.print("[ERROR]: Not enouth masters!\n")
      return
    end
    masters.each do |m|
      _r = redis.get(m.hostname)
      redisVectors.append(_r) unless _r.nil?
    end
    redisVectors.each do |v|
      _s = JSON.parse(v)
      s = _s["status"]
      STDERR.print "#{s}\n"
    end
  end

  desc "Get vectors from masters"
  task masters: :environment do
    error = nil
    settings = Rails.application.config.atella  
    begin
      mastersConfig = TOML.load_file(settings["atella"]["masterServersConfig"])
      securityConfig = TOML.load_file(settings["atella"]["securityConfig"])
    rescue => v
      @error = v
    end

    if error 
      STDERR.print("[FATAL]: #{error}\n") 
      return 
    end
    
    redis = Redis.new(host: settings["atella"]["redisHost"])
    masters = Host.where(:is_master => true)
    if masters.nil? || redis.nil?
      STDERR.print("[FATAL]: redis - #{redis.nil?}, masters - #{masters.nil?}\n") 
      return
    end

    masters.each do |m|
      vector = wrap_master_host(m.address, m.hostname, securityConfig)
      vector = processVector(vector)
      redisVector = redis.get(m.hostname)
      unless vector.eql?(redisVector)
        redis.set(m.hostname, vector)
        redisDataPretty = JSON.pretty_generate(JSON.parse(vector))
        unless redisDataPretty.nil?
          redisDataPretty.gsub!('\\', '')
        end
        ActionCable.server.broadcast("Notifications", { action: "tagUpdate", tagId: "#{m.hostname}_content", content: redisDataPretty})
      end
    end
  end
end
