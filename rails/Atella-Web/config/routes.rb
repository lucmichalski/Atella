Rails.application.routes.draw do
  # For details on the DSL available within this file, see https://guides.rubyonrails.org/routing.html

  
  #get
  get '/pkg' => "atella_main#pkg"
  get '/cfg' => "atella_main#cfg"
  get '/sectors' => "atella_main#sectors"
  get '/hosts' => "atella_main#hosts"
  
  #post
  post '/pkg/:pkg/:act' => "atella_main#pkg_post"
  
  #root
  root "atella_main#atella"

  # #404
  get '/*permalink' => 'atella_main#render_404'
end
