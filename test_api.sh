  #!/bin/bash                                                                                                                             
  set -e
  BASE="http://localhost:8080"
                                                                                                                                            
  #echo "→ Signup"
  curl -s -X POST $BASE/v1/signup \                                                                                                         
    -H "Content-Type: application/json" \                                                                                                 
    -d '{"email":"flow@test.com","password":"password123"}' | jq                                                                            
  
  #echo "→ Login"                                                                                                                            
  TOKEN=$(curl -s -X POST $BASE/v1/login \                                                                                                
    -H "Content-Type: application/json" \                                                                                                   
    -d '{"email":"flow@test.com","password":"password123"}' \
    | jq -r '.data')                                                                                                                        
  echo "Token: ${TOKEN:0:30}..."                                                                                                          
                                                                                                                                            
  echo "→ Create person"                                                                                                                    
  curl -s -X POST $BASE/v1/persons/create \                                                                                               
    -H "Content-Type: application/json" \                                                                                                   
    -H "Authorization: Bearer $TOKEN" \                                                                                                   
    -d '{"name":"Ana","age":25,"communities":[{"name":"QA"}]}' | jq                                                                         
  
  echo "→ Get all"                                                                                                                          
  curl -s -X GET $BASE/v1/persons/get-all \                                                                                               
    -H "Authorization: Bearer $TOKEN" | jq                                                                                                  
  
  echo "→ Get by ID 1"                                                                                                                      
  curl -s -X GET $BASE/v1/persons/1 \                                                                                                     
    -H "Authorization: Bearer $TOKEN" | jq                                                                                                  
                                                                                                                                          
  echo "→ Update ID 1"                                                                                                                      
  curl -s -X PUT $BASE/v1/persons/1 \
    -H "Content-Type: application/json" \                                                                                                   
    -H "Authorization: Bearer $TOKEN" \                                                                                                   
    -d '{"name":"Ana López","age":26,"communities":[{"name":"QA"}]}' | jq                                                                   
  
  echo "→ Delete ID 1"                                                                                                                      
  curl -s -X DELETE $BASE/v1/persons/1 \                                                                                                  
    -H "Authorization: Bearer $TOKEN" | jq  