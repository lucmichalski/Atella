Rails.application.routes.draw do
  # For details on the DSL available within this file, see https://guides.rubyonrails.org/routing.html

  #root
  root "atella_main#atella"

  #get
  get '/pkg' => "atella_main#pkg"
  get '/cfg' => "atella_main#cfg"
  get '/sectors' => "atella_main#sectors"
  get '/hosts' => "atella_main#hosts"
  
  #404
  get '/*permalink' => 'atella_main#render_404'
end
