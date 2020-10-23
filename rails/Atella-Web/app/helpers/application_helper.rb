module ApplicationHelper
  def wrap_master_host(addr, name, sec)
    status = Hash.new
    status["status"] = "Failed"
    status["version"] = "unknown"
    if sec["security"].nil?
      return status.to_json
    end
    if sec["security"]["code"].nil?
      return status.to_json
    end
    begin
      s = TCPSocket.open(addr, 5223)
      s.puts(sec["security"]["code"])
      loop do
        line = s.gets
        break if (line == "Bye!" || line.nil?)
        l = line.rstrip.split
        case (l[0])
          when "Bye!"
            break
          when "canTalk"
            s.puts("export master")
            st = s.gets.split[1]
            status["status"] = JSON.parse(st)
            s.puts("version")
            status["version"] = s.gets.split[1]
            unless status["version"].nil?
              status["version"].rstrip!
            end
            s.puts("exit")
            break
          else
            s.puts("exit")
            break
        end
      end
      s.close
    rescue => v
      status["status"] = v
    end
    return status.to_json
  end

  def processVector(v)
    _s = JSON.parse(v)
    s = _s["status"]
    # s.each do |i|
    #   print "#{i}\n"
    # end
    return v
  end
end