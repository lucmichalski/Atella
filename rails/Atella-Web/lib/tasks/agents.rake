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
    error = nil
    settings = Rails.application.config.atella
    
    redis = Redis.new(host: settings["atella"]["redisHost"])
    masters = Host.where(:is_master => true)
    hosts = Host.all

    if masters.nil? || redis.nil?
      STDERR.print("[FATAL]: redis - #{redis.nil?}, masters - #{masters.nil?}\n") 
      return
    end

    statusVector = Hash.new
    redisVector = Array.new

    masters.each do |m|
      r = redis.get(m.hostname)
      redisData = nil
      redisData = JSON.parse(r) unless r.nil?
      redisVector << redisData["status"] unless (redisData.nil? || redisData["status"].nil?)
    end

    redisVector.each do |master|
      master.each do |client|
        client[1].each do |vec|
          statusVector[vec["hostname"]] = true if statusVector[vec["hostname"]].nil?
          statusVector[vec["hostname"]] = vec["status"] && statusVector[vec["hostname"]]
        end
      end
    end

    hosts.each do |h| 
      statusVector[h.hostname] = false if statusVector[h.hostname].nil?
      h.status = statusVector[h.hostname]
      h.save
    end
  end

  desc "Get vectors from masters"
  task masters: :environment do
    error = nil
    settings = Rails.application.config.atella  
    begin
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
