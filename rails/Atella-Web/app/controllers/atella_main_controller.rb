class AtellaMainController < ApplicationController
  def initialize
    super
  end

  def atella
    begin
      mastersConfig = TOML.load_file(@settings["atella"]["masterServersConfig"])
      securityConfig = TOML.load_file(@settings["atella"]["securityConfig"])
    rescue => v
      @error = v
    end
    if @error.nil?
      @redis = Redis.new(url: "redis://#{ENV.fetch("REDIS_URL") { "127.0.0.1" }}")
      @masters = Host.where(:is_master => true)
      
      if @masters.nil?
        @error = "Not enouth masters!"
      end
    end
  end

  def sectors
    begin
      @sectorsConfig = TOML.load_file(@settings["atella"]["sectorsConfig"])
    rescue => v
      @error = v
    end
  end

  def hosts
    @hostsConfig = Host.all
  end
  
  def pkg
    pkgDir = "#{@settings["atella"]["filesDirectory"] + @settings["atella"]["packagesSubDirectory"]}"
    @debPkgDir = Dir["#{pkgDir}deb/*.deb"].sort
    @rpmPkgDir = Dir["#{pkgDir}rpm/*.rpm"].sort
    @tarPkgDir = Dir["#{pkgDir}tar/*.tar*"].sort
  end
  
  def pkg_post
    act = params[:act]
    pkg = params[:pkg]
    pkgDir = "#{@settings["atella"]["filesDirectory"] + @settings["atella"]["packagesSubDirectory"]}"
    post = params["Delete"]
    case act
    when "delete"
      f = "#{pkgDir}#{pkg}/#{post}"
      File.delete(f) if File.exist?(f)
    end
    redirect_to pkg_path
  end

  def cfg
    dir = "#{@settings["atella"]["filesDirectory"] + @settings["atella"]["configsSubDirectory"]}"
    @cfgDir = Dir["#{dir}*"].sort
  end

  def dev
    # render file: "#{Rails.root}/public/403", status: 403
    if ENV['RAILS_ENV'].eql?('production') 
      render file: "#{Rails.root}/public/403", status: 403
    end
  end
  
  def render_404
    render file: "#{Rails.root}/public/404", status: :not_found
  end
end
